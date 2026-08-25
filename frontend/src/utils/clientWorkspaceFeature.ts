export type ClientWorkspaceRuntimeKind = 'personal' | 'enterprise'

export function shouldShowWorkspaceBinding(
  kind: ClientWorkspaceRuntimeKind,
  enabled: boolean,
): boolean {
  return kind !== 'enterprise' || enabled === true
}

export function buildWorkspaceBindingPayload(
  kind: ClientWorkspaceRuntimeKind,
  enabled: boolean,
  selectedWorkspaceId: string | null | undefined,
): Record<string, unknown> {
  const selected = String(selectedWorkspaceId || '').trim()
  if (!selected || !shouldShowWorkspaceBinding(kind, enabled)) return {}

  return {
    workspace_binding: kind === 'enterprise'
      ? { workspace_id: selected }
      : { root_path: selected },
  }
}
