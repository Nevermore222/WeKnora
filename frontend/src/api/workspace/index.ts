import { get, post } from '@/utils/request';

export type WorkspaceStatus = 'available' | 'missing' | 'access_denied' | 'archived';

export interface WorkspaceEntry {
  id: string;
  name: string;
  relative_path: string;
  status: WorkspaceStatus;
}

export async function listWorkspaces() {
  return get('/api/v1/workspaces');
}

export async function createWorkspace(name: string) {
  return post('/api/v1/workspaces', { name });
}
