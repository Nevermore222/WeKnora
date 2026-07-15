import {
  fetchWorkspaceFileBlob,
  workspaceFileDownloadUrl,
  workspaceFileRawUrl,
  type WorkspaceFileEntry,
} from '@/api/workspace';

export type PreviewFileSource = 'chat-artifact' | 'workspace-file';
export type PreviewKind = 'markdown' | 'text' | 'image' | 'pdf' | 'spreadsheet' | 'presentation' | 'archive' | 'other';

export interface PreviewFileRef {
  source: PreviewFileSource;
  name: string;
  path: string;
  relativePath?: string;
  mimeType?: string;
  kind?: PreviewKind;
  size?: number;
  modifiedAt?: string;
  previewUrl?: string;
  rawUrl?: string;
  downloadUrl?: string;
  workspaceId?: string;
  sessionId?: string;
  jobId?: string;
}

export function detectPreviewKind(name: string, mimeType = ''): PreviewKind {
  const lower = name.toLowerCase();
  if (lower.endsWith('.md') || lower.endsWith('.markdown')) return 'markdown';
  if (lower.endsWith('.txt') || lower.endsWith('.log') || lower.endsWith('.json') || lower.endsWith('.csv')) return 'text';
  if (mimeType.startsWith('image/') || /\.(png|jpe?g|gif|webp|svg|bmp)$/i.test(lower)) return 'image';
  if (mimeType === 'application/pdf' || lower.endsWith('.pdf')) return 'pdf';
  if (/\.(xlsx?)$/i.test(lower)) return 'spreadsheet';
  if (/\.(pptx?|key)$/i.test(lower)) return 'presentation';
  if (/\.(zip|tar|gz|7z|rar)$/i.test(lower)) return 'archive';
  return 'other';
}

export function canInlinePreview(ref: PreviewFileRef) {
  const kind = ref.kind || detectPreviewKind(ref.name, ref.mimeType);
  return kind === 'markdown' || kind === 'text' || kind === 'image' || kind === 'pdf';
}

export function normalizeWorkspaceFileRef(workspaceId: string, file: WorkspaceFileEntry): PreviewFileRef {
  const path = file.relative_path;
  const kind = (file.kind as PreviewKind) || detectPreviewKind(file.name);
  return {
    source: 'workspace-file',
    name: file.name,
    path,
    relativePath: path,
    kind,
    size: file.size,
    modifiedAt: file.modified_at,
    workspaceId,
    previewUrl: `/api/v1/workspaces/${encodeURIComponent(workspaceId)}/files/preview?path=${encodeURIComponent(path)}`,
    rawUrl: workspaceFileRawUrl(workspaceId, path),
    downloadUrl: workspaceFileDownloadUrl(workspaceId, path),
  };
}

export function normalizeChatArtifact(artifact: Record<string, any>): PreviewFileRef {
  const path = artifact.relative_path || artifact.relativePath || artifact.path || artifact.name || '';
  const workspaceId = artifact.workspace_id || artifact.workspaceId;
  const name = artifact.name || String(path).split('/').pop() || String(path);
  return {
    source: 'chat-artifact',
    name,
    path,
    relativePath: path,
    kind: artifact.kind || detectPreviewKind(path, artifact.content_type || artifact.mimeType || ''),
    mimeType: artifact.content_type || artifact.mimeType,
    size: artifact.size,
    workspaceId,
    sessionId: artifact.session_id || artifact.sessionId,
    jobId: artifact.job_id || artifact.jobId,
    previewUrl: workspaceId ? `/api/v1/workspaces/${encodeURIComponent(workspaceId)}/files/preview?path=${encodeURIComponent(path)}` : undefined,
    rawUrl: workspaceId ? workspaceFileRawUrl(workspaceId, path) : undefined,
    downloadUrl: workspaceId ? workspaceFileDownloadUrl(workspaceId, path) : undefined,
  };
}

export function fileInfoText(ref: PreviewFileRef) {
  return [
    `Name: ${ref.name}`,
    `Type: ${ref.kind || detectPreviewKind(ref.name, ref.mimeType)}`,
    ref.size != null ? `Size: ${ref.size} bytes` : null,
    `Path: ${ref.relativePath || ref.path}`,
    ref.workspaceId ? `Workspace: ${ref.workspaceId}` : null,
    ref.sessionId ? `Session: ${ref.sessionId}` : null,
    ref.jobId ? `Job: ${ref.jobId}` : null,
  ].filter(Boolean).join('\n');
}

function revokeObjectUrlLater(url: string) {
  window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
}

export async function openPreviewFile(ref: PreviewFileRef) {
  if (!ref.workspaceId) {
    throw new Error('当前文件缺少 workspaceId，无法打开');
  }
  const target = window.open('about:blank', '_blank', 'noopener,noreferrer');
  if (!target) {
    throw new Error('浏览器阻止了新窗口，请允许弹窗后重试');
  }
  try {
    const blob = await fetchWorkspaceFileBlob(ref.workspaceId, ref.relativePath || ref.path, 'raw');
    const objectUrl = URL.createObjectURL(blob);
    target.location.href = objectUrl;
    revokeObjectUrlLater(objectUrl);
  } catch (error) {
    target.close();
    throw error;
  }
}

export async function downloadPreviewFile(ref: PreviewFileRef) {
  if (!ref.workspaceId) {
    throw new Error('当前文件缺少 workspaceId，无法下载');
  }
  const blob = await fetchWorkspaceFileBlob(ref.workspaceId, ref.relativePath || ref.path, 'download');
  const objectUrl = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = objectUrl;
  link.download = ref.name || 'download';
  document.body.appendChild(link);
  link.click();
  link.remove();
  revokeObjectUrlLater(objectUrl);
}
