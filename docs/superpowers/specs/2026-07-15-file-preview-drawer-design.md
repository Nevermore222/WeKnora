# File Preview Drawer Design

Date: 2026-07-15

Status: Approved for implementation planning

## Goal

Add a unified file preview sidebar for Xelora-generated artifacts and workspace files. The first version should make text, Markdown, images, and PDFs previewable from both chat and workspace entry points, while preserving safe download and copy actions for all file types.

The interaction should follow mature file-preview patterns from systems such as Cursor, VS Code, cloud-drive side panels, and developer workspace tools: a persistent right-side preview surface, clear file metadata, quick actions, and an "open with" style menu. Implementation should reuse only license-compatible open-source code. If a reference system is proprietary or license-incompatible, reproduce the behavior and information architecture rather than copying source.

## Scope

### In Scope

- A global `FilePreviewDrawer` that can be opened from chat artifacts and workspace file rows.
- A shared frontend `filePreview` store or equivalent state module.
- A normalized `PreviewFileRef` model consumed by both entry points.
- Inline preview for Markdown, plain text, images, and PDF.
- Unsupported preview state for Office and other files, with download and copy actions still available.
- Safe "open with" menu actions that do not launch local desktop applications.
- Backend workspace file preview/raw/download endpoints if existing APIs are insufficient.
- Workspace path boundary checks on every backend file endpoint.
- Docker rebuild and local deployment verification for `app` and `frontend`.

### Out Of Scope

- Directly launching Cursor, VS Code, Terminal, WSL, IntelliJ IDEA, or default desktop applications from the browser.
- Exposing host absolute paths to remote users.
- Full Office document preview for DOCX, XLSX, and PPTX.
- Persistent artifact database redesign.
- Multi-file comparison, diff view, or collaborative editing.

## User Experience

The preview opens as a right-side drawer over the current page. It keeps the user in context instead of navigating away from the chat or workspace.

Drawer header:

- File name.
- File type badge.
- File size when available.
- Source label such as `Chat artifact` or `Workspace file`.
- Workspace/session context when available.

Drawer body:

- Markdown/text: render readable content using the existing Markdown pipeline where appropriate; fallback to monospaced plain text for unsupported text formats.
- Image: fit-to-panel image preview with scroll/zoom-ready layout.
- PDF: browser-native embedded preview using the raw file URL.
- Unsupported files: show a clear placeholder with file metadata and available actions.

Actions:

- Preview: enabled when inline preview is supported, disabled otherwise.
- Download: uses the backend attachment endpoint.
- Copy path: copies the workspace-relative path only.
- Copy file info: copies file name, type, size, workspace-relative path, and source.
- Open containing folder: first version shows guidance and optionally copies the relative folder path. It must not call `file://` or launch a local process.

## Data Model

Frontend file references are normalized into `PreviewFileRef`:

```ts
export type PreviewFileSource = 'chat-artifact' | 'workspace-file';

export interface PreviewFileRef {
  source: PreviewFileSource;
  name: string;
  path: string;
  relativePath?: string;
  mimeType?: string;
  kind?: 'markdown' | 'text' | 'image' | 'pdf' | 'spreadsheet' | 'presentation' | 'archive' | 'other';
  size?: number;
  modifiedAt?: string;
  previewUrl?: string;
  rawUrl?: string;
  downloadUrl?: string;
  workspaceId?: string;
  sessionId?: string;
  jobId?: string;
}
```

Chat artifacts should be adapted from existing executor artifact fields such as `relative_path`, `kind`, `preview_state`, `workspace_id`, `session_id`, and `job_id`.

Workspace files should be adapted from workspace file list rows. If the current workspace API does not expose file rows, add the minimum list/read endpoints needed for this feature.

## Backend API

Use existing API names if compatible. Otherwise add these minimal endpoints:

- `GET /api/v1/workspaces/:id/files`
- `GET /api/v1/workspaces/:id/files/preview?path=...`
- `GET /api/v1/workspaces/:id/files/raw?path=...`
- `GET /api/v1/workspaces/:id/files/download?path=...`

Rules:

- `path` is always workspace-relative.
- The backend resolves the requested path under the workspace root and rejects path traversal or symlink escapes.
- `preview` returns text for Markdown/plain text only and enforces a size cap, initially 1 MB.
- `raw` is embeddable for images and PDFs and returns the correct content type.
- `download` forces `Content-Disposition: attachment`.
- Unsupported preview types return a structured unsupported response or a clear HTTP error that the frontend can render as the unsupported state.

## Frontend Components

Recommended structure:

```text
frontend/src/stores/filePreview.ts
frontend/src/components/file-preview/FilePreviewDrawer.vue
frontend/src/components/file-preview/FilePreviewActions.vue
frontend/src/components/file-preview/FilePreviewBody.vue
frontend/src/components/file-preview/previews/TextPreview.vue
frontend/src/components/file-preview/previews/ImagePreview.vue
frontend/src/components/file-preview/previews/PdfPreview.vue
frontend/src/components/file-preview/previews/UnsupportedPreview.vue
frontend/src/api/workspace-files/index.ts
frontend/src/utils/filePreview.ts
```

Entry points:

- Chat artifact chips/cards call `filePreviewStore.open(ref)`.
- Workspace file rows call the same `open(ref)`.
- The drawer is mounted once near the app shell so both entry points share the same state.

## Security

- Do not expose host absolute paths in UI actions intended for normal users.
- Do not implement browser-triggered local application launches in v1.
- Do not use `file://` links.
- Enforce workspace path boundaries on the backend, not only in the frontend.
- Return safe, explicit errors for missing, inaccessible, too-large, or unsupported files.

## Testing

Backend tests:

- Workspace file list only returns files inside the workspace.
- `../secret.txt` and equivalent path traversal attempts are rejected.
- Text preview enforces the size cap.
- Raw image/PDF endpoints return appropriate content types.
- Download endpoint returns attachment disposition.

Frontend tests:

- `PreviewFileRef` normalization maps chat artifacts and workspace file rows consistently.
- File kind detection selects the correct preview component.
- Unsupported files still show download/copy actions.
- Menu actions copy relative path and file info correctly.

Integration verification:

- Create or place Markdown/text, image, PDF, and Office files in a bound workspace.
- Open Markdown/text, image, and PDF from workspace file list and verify the drawer renders them.
- Open the same supported types from chat artifact output and verify the same drawer renders them.
- Verify Office files render unsupported preview state with download/copy actions.
- Verify path traversal is rejected through direct HTTP requests.
- Rebuild and redeploy Docker `app` and `frontend`; confirm services are healthy and registration still works.

## Deployment Plan

Use the current local Docker workflow:

1. Build updated `app` image if backend APIs changed.
2. Build updated `frontend` image for drawer UI changes.
3. Deploy in the local offline compose directory with `docker compose up -d app frontend`.
4. Verify `docker compose ps` health.
5. Run HTTP checks for health, file preview APIs, and registration.
6. Use the browser UI to verify drawer behavior from both entry points.

The existing AES key length and migration BOM fixes remain prerequisites for stable local deployment.

## Future Extensions

- Office preview via conversion service or embedded office renderer.
- Local deployment "open with" provider behind an explicit trusted local agent.
- Configurable desktop open targets such as Cursor, VS Code, Terminal, WSL, Git Bash, and IntelliJ IDEA.
- Persistent artifact sidebar/history per session.
- File search and quick preview keyboard shortcuts.
