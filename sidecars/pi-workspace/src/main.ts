import readline from 'node:readline'
import { PROTOCOL_VERSION, type HandshakeIn, type HandshakeOut, type Request, type Response } from './protocol.js'
import { handle } from './tools.js'

function write(obj: unknown): void {
  process.stdout.write(`${JSON.stringify(obj)}\n`)
}

async function onLine(line: string): Promise<void> {
  let msg: Record<string, unknown>
  try {
    msg = JSON.parse(line) as Record<string, unknown>
  } catch (err) {
    const response: Response = {
      ok: false,
      error_code: 'bad_json',
      error_message: err instanceof Error ? err.message : String(err),
    }
    write(response)
    return
  }

  if (msg.type === 'handshake') {
    const hs = msg as unknown as HandshakeIn
    const out: HandshakeOut =
      hs.protocol_version === PROTOCOL_VERSION
        ? { type: 'handshake_ok', protocol_version: PROTOCOL_VERSION, ok: true }
        : {
            type: 'handshake_ok',
            protocol_version: PROTOCOL_VERSION,
            ok: false,
            error_code: 'protocol_mismatch',
          }
    write(out)
    return
  }

  const req = msg as unknown as Request
  const resp = await handle(req)
  write(resp)
}

const rl = readline.createInterface({ input: process.stdin, crlfDelay: Infinity })
rl.on('line', (line) => {
  void onLine(line)
})
rl.on('close', () => {
  process.exit(0)
})
