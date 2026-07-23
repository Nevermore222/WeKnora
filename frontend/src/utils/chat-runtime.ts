export function resolveAgentModeEnabled(
  agentId: string,
  agentMode: string | null | undefined,
  currentEnabled: boolean,
): boolean {
  if (agentId === 'builtin-quick-answer') return false
  if (agentId === 'builtin-smart-reasoning') return true
  if (agentMode === 'smart-reasoning') return true
  if (agentMode === 'quick-answer') return false
  return currentEnabled
}

export function resolveContinuableAssistantMessageId(message: {
  id?: unknown
  role?: unknown
  is_completed?: unknown
} | null | undefined): string | null {
  if (!message || message.role !== 'assistant' || message.is_completed === true) {
    return null
  }
  const id = typeof message.id === 'string' ? message.id.trim() : ''
  return id || null
}

export function resolveStreamApiBaseUrl(
  desktopBackendUrl: string | null | undefined,
  webBaseUrl: string,
): string {
  const desktopBase = String(desktopBackendUrl || '').trim().replace(/\/+$/, '')
  return desktopBase || webBaseUrl
}
