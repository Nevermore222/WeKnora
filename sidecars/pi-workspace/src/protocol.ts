export const PROTOCOL_VERSION = 1

export type Request = {
  protocol_version: number
  correlation_id?: string
  session_id?: string
  tool_name: string
  args?: unknown
  cwd: string
  permission_mode?: string
  timeout_ms?: number
}

export type Response = {
  ok: boolean
  error_code?: string
  error_message?: string
  content?: string
  artifacts?: string[]
  duration_ms?: number
  data?: Record<string, unknown>
}

export type HandshakeIn = {
  type: 'handshake'
  protocol_version: number
}

export type HandshakeOut = {
  type: 'handshake_ok'
  protocol_version: number
  ok: boolean
  error_code?: string
}
