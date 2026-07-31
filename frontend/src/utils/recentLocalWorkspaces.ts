export type RecentLocalWorkspace = {
  workspace_id: string
  workspace_name: string
  root_path: string
  last_used_at: string
  stale?: boolean
}

const KEY = 'xelora_recent_local_workspaces'
const MAX = 20

export function loadRecentLocalWorkspaces(): RecentLocalWorkspace[] {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (item) =>
        item &&
        typeof item.root_path === 'string' &&
        typeof item.workspace_name === 'string',
    ) as RecentLocalWorkspace[]
  } catch {
    return []
  }
}

export function rememberRecentLocalWorkspace(
  entry: Omit<RecentLocalWorkspace, 'last_used_at' | 'stale'>,
): void {
  const now = new Date().toISOString()
  const next: RecentLocalWorkspace[] = [
    { ...entry, last_used_at: now },
    ...loadRecentLocalWorkspaces().filter(
      (item) => item.root_path !== entry.root_path && item.workspace_id !== entry.workspace_id,
    ),
  ].slice(0, MAX)
  localStorage.setItem(KEY, JSON.stringify(next))
}

export function dismissRecentLocalWorkspace(workspaceIdOrPath: string): void {
  const next = loadRecentLocalWorkspaces().filter(
    (item) => item.workspace_id !== workspaceIdOrPath && item.root_path !== workspaceIdOrPath,
  )
  localStorage.setItem(KEY, JSON.stringify(next))
}

export function markRecentLocalWorkspaceStale(rootPath: string): void {
  const next = loadRecentLocalWorkspaces().map((item) =>
    item.root_path === rootPath ? { ...item, stale: true } : item,
  )
  localStorage.setItem(KEY, JSON.stringify(next))
}
