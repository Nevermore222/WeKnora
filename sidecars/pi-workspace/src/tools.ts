import { spawn } from 'node:child_process'
import fs from 'node:fs/promises'
import path from 'node:path'
import type { Request, Response } from './protocol.js'

class ToolError extends Error {
  code: string
  constructor(code: string, message: string) {
    super(message)
    this.code = code
  }
}

function resolveInCwd(cwd: string, rel: string): string {
  const root = path.resolve(cwd)
  const abs = path.resolve(root, rel || '.')
  const relToRoot = path.relative(root, abs)
  if (relToRoot.startsWith('..') || path.isAbsolute(relToRoot)) {
    throw new ToolError('path_out_of_workspace', `path escapes cwd: ${rel}`)
  }
  return abs
}

type Args = Record<string, unknown>

function asArgs(raw: unknown): Args {
  if (raw && typeof raw === 'object' && !Array.isArray(raw)) {
    return raw as Args
  }
  return {}
}

async function workspaceRead(cwd: string, args: Args): Promise<Response> {
  const rel = String(args.path ?? '')
  const abs = resolveInCwd(cwd, rel)
  const content = await fs.readFile(abs, 'utf8')
  return { ok: true, content, artifacts: [rel] }
}

async function workspaceWrite(cwd: string, args: Args): Promise<Response> {
  const rel = String(args.path ?? '')
  const content = String(args.content ?? '')
  const abs = resolveInCwd(cwd, rel)
  await fs.mkdir(path.dirname(abs), { recursive: true })
  await fs.writeFile(abs, content, 'utf8')
  return { ok: true, content: `wrote ${rel}`, artifacts: [rel] }
}

async function workspaceEdit(cwd: string, args: Args): Promise<Response> {
  const rel = String(args.path ?? '')
  const oldText = String(args.old_text ?? '')
  const newText = String(args.new_text ?? '')
  const abs = resolveInCwd(cwd, rel)
  const current = await fs.readFile(abs, 'utf8')
  if (!current.includes(oldText)) {
    throw new ToolError('edit_failed', 'old_text not found')
  }
  const next = current.replace(oldText, newText)
  await fs.writeFile(abs, next, 'utf8')
  return { ok: true, content: `edited ${rel}`, artifacts: [rel] }
}

async function workspaceList(cwd: string, args: Args): Promise<Response> {
  const rel = String(args.path ?? '.')
  const abs = resolveInCwd(cwd, rel)
  const entries = await fs.readdir(abs, { withFileTypes: true })
  const lines = entries.map((e) => `${e.isDirectory() ? 'dir' : 'file'}\t${e.name}`)
  return { ok: true, content: lines.join('\n'), artifacts: [rel] }
}

async function workspaceShell(cwd: string, args: Args, timeoutMs: number): Promise<Response> {
  const command = String(args.command ?? '')
  if (!command.trim()) {
    throw new ToolError('invalid_args', 'command required')
  }
  const root = path.resolve(cwd)
  return await new Promise<Response>((resolve) => {
    const child = spawn(command, {
      cwd: root,
      shell: true,
      windowsHide: true,
    })
    let stdout = ''
    let stderr = ''
    const timer = setTimeout(() => {
      child.kill()
      resolve({
        ok: false,
        error_code: 'timeout',
        error_message: `shell timed out after ${timeoutMs}ms`,
      })
    }, timeoutMs)
    child.stdout.on('data', (d) => {
      stdout += String(d)
    })
    child.stderr.on('data', (d) => {
      stderr += String(d)
    })
    child.on('error', (err) => {
      clearTimeout(timer)
      resolve({
        ok: false,
        error_code: 'runtime_unavailable',
        error_message: err.message,
      })
    })
    child.on('close', (code) => {
      clearTimeout(timer)
      const content = [stdout, stderr].filter(Boolean).join('\n')
      if (code !== 0) {
        resolve({
          ok: false,
          error_code: 'shell_failed',
          error_message: content || `exit ${code}`,
          content,
        })
        return
      }
      resolve({ ok: true, content })
    })
  })
}

async function executeSkillScript(cwd: string, args: Args, timeoutMs: number): Promise<Response> {
  // script_abs is trusted only when injected by the Go Facade after skill prepare.
  const scriptAbs = String(args.script_abs ?? '')
  const script = String(args.script ?? args.script_path ?? '')
  let abs = ''
  if (scriptAbs.trim()) {
    abs = path.resolve(scriptAbs)
    await fs.access(abs)
  } else if (script.trim()) {
    abs = resolveInCwd(cwd, script)
  } else {
    throw new ToolError('invalid_args', 'script or script_abs required')
  }
  const extra = Array.isArray(args.args) ? args.args.map(String) : []
  const isJs = /\.[cm]?jsx?$/i.test(abs) || /\.ts$/i.test(abs)
  const isPy = /\.py$/i.test(abs)
  let command: string
  if (isJs) {
    command = `node ${JSON.stringify(abs)} ${extra.map((a) => JSON.stringify(a)).join(' ')}`
  } else if (isPy) {
    command = `python ${JSON.stringify(abs)} ${extra.map((a) => JSON.stringify(a)).join(' ')}`
  } else if (process.platform === 'win32') {
    command = `powershell -NoProfile -ExecutionPolicy Bypass -File ${JSON.stringify(abs)} ${extra.map((a) => JSON.stringify(a)).join(' ')}`
  } else {
    command = `bash ${JSON.stringify(abs)} ${extra.map((a) => JSON.stringify(a)).join(' ')}`
  }
  return workspaceShell(cwd, { command }, timeoutMs)
}

export async function handle(req: Request): Promise<Response> {
  const started = Date.now()
  const cwd = req.cwd
  if (!cwd || typeof cwd !== 'string') {
    return { ok: false, error_code: 'binding_invalid', error_message: 'cwd required' }
  }
  const args = asArgs(req.args)
  const timeoutMs = req.timeout_ms && req.timeout_ms > 0 ? req.timeout_ms : 60_000
  try {
    let resp: Response
    switch (req.tool_name) {
      case 'workspace_read':
        resp = await workspaceRead(cwd, args)
        break
      case 'workspace_write':
        resp = await workspaceWrite(cwd, args)
        break
      case 'workspace_edit':
        resp = await workspaceEdit(cwd, args)
        break
      case 'workspace_list':
        resp = await workspaceList(cwd, args)
        break
      case 'workspace_shell':
        resp = await workspaceShell(cwd, args, timeoutMs)
        break
      case 'execute_skill_script':
        resp = await executeSkillScript(cwd, args, timeoutMs)
        break
      default:
        return {
          ok: false,
          error_code: 'unknown_tool',
          error_message: `unknown tool: ${req.tool_name}`,
        }
    }
    resp.duration_ms = Date.now() - started
    return resp
  } catch (err) {
    if (err instanceof ToolError) {
      return {
        ok: false,
        error_code: err.code,
        error_message: err.message,
        duration_ms: Date.now() - started,
      }
    }
    const message = err instanceof Error ? err.message : String(err)
    return {
      ok: false,
      error_code: 'runtime_unavailable',
      error_message: message,
      duration_ms: Date.now() - started,
    }
  }
}
