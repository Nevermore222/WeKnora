export interface WorkspaceSelectionEntry {
  id: string;
  status: string;
}

export function restoreWorkspaceSelection(
  savedId: string | null | undefined,
  entries: WorkspaceSelectionEntry[],
): string {
  if (!savedId) return '';
  return entries.some((entry) => entry.id === savedId && entry.status === 'available') ? savedId : '';
}
