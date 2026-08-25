import type { RuntimeContext } from './api-context.ts'
import type { DesktopServerProfile } from '@/api/desktop-remote'
import { brandDisplayName } from '../config/brandName.ts'

export interface DefaultEnterpriseBootstrap {
  default_enterprise_server_url?: string
  default_enterprise_server_fallback_url?: string
  default_enterprise_server_name?: string
  default_enterprise_allow_insecure?: boolean
}

export interface DefaultEnterpriseBootstrapDeps {
  ensureProfile: (config: {
    name?: string
    base_url: string
    allow_insecure_transport?: boolean
  }) => Promise<DesktopServerProfile>
  setContext: (context: RuntimeContext) => void
  delay?: (ms: number) => Promise<void>
}

export interface DefaultEnterpriseBootstrapOptions {
  attempts?: number
  delayMs?: number
}

export type DefaultEnterpriseConnectionFailureKind =
  | 'unreachable'
  | 'unauthorized'
  | 'incompatible'
  | 'invalid'
  | 'unknown'

export function getDefaultEnterpriseErrorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  if (typeof error === 'string') return error
  if (!error || typeof error !== 'object') return ''

  const record = error as Record<string, unknown>
  for (const key of ['message', 'error', 'detail']) {
    const value = record[key]
    if (typeof value === 'string' && value.trim()) return value
    if (value && typeof value === 'object') {
      const nestedMessage = (value as Record<string, unknown>).message
      if (typeof nestedMessage === 'string' && nestedMessage.trim()) return nestedMessage
    }
  }

  return ''
}

export function classifyDefaultEnterpriseConnectionError(
  error: unknown,
): DefaultEnterpriseConnectionFailureKind {
  const record = error && typeof error === 'object' ? error as Record<string, unknown> : null
  const response = record?.response && typeof record.response === 'object'
    ? record.response as Record<string, unknown>
    : null
  const rawStatus = record?.status ?? response?.status
  const status = typeof rawStatus === 'number' ? rawStatus : Number(rawStatus)
  const message = getDefaultEnterpriseErrorMessage(error).toLowerCase()

  if (status === 409 || /api contract|incompatible.*(api|contract)|unsupported.*version/.test(message)) {
    return 'incompatible'
  }
  if (status === 401 || status === 403 || /unauthorized|forbidden|access denied/.test(message)) {
    return 'unauthorized'
  }
  if (
    status === 400 ||
    /invalid.*(url|origin|address|target)|http\(s\) origin|remote target/.test(message)
  ) {
    return 'invalid'
  }
  if (
    status >= 500 ||
    /dial tcp|no such host|connection refused|econnrefused|enotfound|network|timeout|timed out|failed to fetch|remote_request_failed|bad gateway/.test(message)
  ) {
    return 'unreachable'
  }
  return 'unknown'
}

export async function initializeDefaultEnterpriseProfile(
  bootstrap: DefaultEnterpriseBootstrap | null | undefined,
  deps: DefaultEnterpriseBootstrapDeps,
  options: DefaultEnterpriseBootstrapOptions = {},
): Promise<RuntimeContext | null> {
  const baseURL = String(bootstrap?.default_enterprise_server_url || '').trim()
  if (!baseURL) return null

  const attempts = Math.max(1, options.attempts ?? 8)
  const delayMs = Math.max(0, options.delayMs ?? 250)
  const wait = deps.delay ?? defaultDelay
  const candidateURLs = [
    baseURL,
    String(bootstrap?.default_enterprise_server_fallback_url || '').trim(),
  ].filter((value, index, values) => value && values.indexOf(value) === index)
  let lastError: unknown = null

  for (const candidateURL of candidateURLs) {
    for (let attempt = 1; attempt <= attempts; attempt += 1) {
      try {
        const profile = await deps.ensureProfile({
          name: String(bootstrap?.default_enterprise_server_name || `${brandDisplayName} Server`),
          base_url: candidateURL,
          allow_insecure_transport: bootstrap?.default_enterprise_allow_insecure === true,
        })
        const context: RuntimeContext = {
          kind: 'enterprise',
          profileId: profile.id,
          userId: null,
          tenantId: null,
          generation: 0,
        }
        deps.setContext(context)
        return context
      } catch (error) {
        lastError = error
        if (attempt < attempts) {
          await wait(delayMs)
        }
      }
    }
  }

  throw toDefaultEnterpriseError(lastError)
}

function toDefaultEnterpriseError(error: unknown): Error {
  if (error instanceof Error) return error
  const message = getDefaultEnterpriseErrorMessage(error).trim()
  return new Error(message || 'default enterprise profile initialization failed')
}

function defaultDelay(ms: number) {
  return new Promise<void>((resolve) => window.setTimeout(resolve, ms))
}
