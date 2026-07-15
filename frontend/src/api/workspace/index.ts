import { get, post } from '@/utils/request';
import { getApiBaseUrl } from '@/utils/api-base';

export type WorkspaceStatus = 'available' | 'missing' | 'access_denied' | 'archived';

export interface WorkspaceEntry {
  id: string;
  name: string;
  relative_path: string;
  status: WorkspaceStatus;
}

export interface WorkspaceFileEntry {
  name: string;
  relative_path: string;
  kind: string;
  size: number;
  modified_at?: string;
  is_dir?: boolean;
}

export interface WorkspaceFilePreviewResponse {
  name: string;
  relative_path: string;
  content_type: string;
  content: string;
  size: number;
}

export async function listWorkspaces() {
  return get('/api/v1/workspaces');
}

export async function createWorkspace(name: string) {
  return post('/api/v1/workspaces', { name });
}

export async function listWorkspaceFiles(workspaceId: string, path = '') {
  const query = path ? `?path=${encodeURIComponent(path)}` : '';
  return get(`/api/v1/workspaces/${encodeURIComponent(workspaceId)}/files${query}`) as Promise<{ files: WorkspaceFileEntry[] }>;
}

export async function previewWorkspaceFile(workspaceId: string, path: string) {
  return get(`/api/v1/workspaces/${encodeURIComponent(workspaceId)}/files/preview?path=${encodeURIComponent(path)}`) as Promise<WorkspaceFilePreviewResponse>;
}

export function workspaceFileRawUrl(workspaceId: string, path: string) {
  return `${getApiBaseUrl()}/api/v1/workspaces/${encodeURIComponent(workspaceId)}/files/raw?path=${encodeURIComponent(path)}`;
}

export function workspaceFileDownloadUrl(workspaceId: string, path: string) {
  return `${getApiBaseUrl()}/api/v1/workspaces/${encodeURIComponent(workspaceId)}/files/download?path=${encodeURIComponent(path)}`;
}
