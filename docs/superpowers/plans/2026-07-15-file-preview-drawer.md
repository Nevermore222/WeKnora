# File Preview Drawer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a unified file preview drawer for chat artifacts and workspace files, supporting inline Markdown/text, image, and PDF preview with safe download/copy actions.

**Architecture:** Add workspace file APIs on the backend, normalize all preview targets into one frontend `PreviewFileRef`, and mount one global `FilePreviewDrawer` in `App.vue`. Chat artifacts and workspace file rows open the same drawer; unsupported files reuse the same action surface without inline preview.

**Tech Stack:** Go/Gin backend, existing `internal/workspace` service, Vue 3 + Pinia + TDesign frontend, existing Axios request wrapper, Docker Compose local deployment.

---

## Current Context

- Source repo: `E:\Xelora\WeKnora`
- Local deployment package: `E:\Xelora\xelora-offline-20260714-011513`
- Design spec: `docs/superpowers/specs/2026-07-15-file-preview-drawer-design.md`
- Current workspace API only supports `GET /workspaces`, `POST /workspaces`, and `GET /workspaces/:id` in `internal/handler/workspace.go`.
- The app shell is `frontend/src/App.vue`; mount the global drawer next to `ManualKnowledgeEditor` and `UploadConfirmHost`.
- Existing frontend workspace API is `frontend/src/api/workspace/index.ts`.
- Existing chat event rendering is concentrated in `frontend/src/views/chat/components/AgentStreamDisplay.vue` and tool-result components under `frontend/src/views/chat/components/tool-results/`.

## File Structure

### Backend

- Modify: `internal/workspace/local.go`
  - Add file listing and safe file resolution methods.
- Modify: `internal/workspace/local_test.go`
  - Add tests for file listing, path traversal rejection, text preview cap, raw file metadata, and download metadata.
- Modify: `internal/types/interfaces/workspace.go`
  - Extend `WorkspaceService` with file methods.
- Modify: `internal/handler/workspace.go`
  - Add `ListFiles`, `PreviewFile`, `RawFile`, and `DownloadFile` handlers.
- Modify: `internal/handler/workspace_test.go`
  - Add handler tests for API status codes, hidden root path, content type, attachment disposition, and traversal rejection.
- Modify: `internal/router/router.go`
  - Register file routes under `/api/v1/workspaces/:id/files`.

### Frontend

- Modify: `frontend/src/api/workspace/index.ts`
  - Add workspace file row types and preview/raw/download URL helpers.
- Create: `frontend/src/utils/filePreview.ts`
  - Add `PreviewFileRef`, file kind detection, chat/workspace normalization, copy text builders.
- Create: `frontend/src/stores/filePreview.ts`
  - Add global drawer state and actions.
- Create: `frontend/src/components/file-preview/FilePreviewDrawer.vue`
- Create: `frontend/src/components/file-preview/FilePreviewActions.vue`
- Create: `frontend/src/components/file-preview/FilePreviewBody.vue`
- Create: `frontend/src/components/file-preview/previews/TextPreview.vue`
- Create: `frontend/src/components/file-preview/previews/ImagePreview.vue`
- Create: `frontend/src/components/file-preview/previews/PdfPreview.vue`
- Create: `frontend/src/components/file-preview/previews/UnsupportedPreview.vue`
- Modify: `frontend/src/App.vue`
  - Mount `FilePreviewDrawer`.
- Modify: a chat artifact rendering location discovered during implementation, likely `frontend/src/views/chat/components/AgentStreamDisplay.vue` or a tool-result component.
  - Convert artifact rows/cards to `PreviewFileRef` and call `filePreviewStore.open(ref)`.
- Modify or create workspace file list UI location.
  - If no workspace file list page exists, add a minimal file list panel inside the current workspace-related surface rather than inventing a new navigation area.

## Task 1: Backend Workspace File Service

**Files:**
- Modify: `internal/types/interfaces/workspace.go`
- Modify: `internal/workspace/local.go`
- Test: `internal/workspace/local_test.go`

- [ ] **Step 1: Write file service tests**

Add tests in `internal/workspace/local_test.go`:

```go
func TestLocalServiceListFilesAndResolvePreview(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "Reports", "summary.md"), "# Summary\n\nHello")
	mustWriteFile(t, filepath.Join(root, "Reports", "image.png"), []byte{0x89, 0x50, 0x4e, 0x47})

	svc := NewLocalService(root)
	entry, err := svc.Create(context.Background(), CreateInput{Name: "Reports"})
	require.NoError(t, err)

	files, err := svc.ListFiles(context.Background(), entry.ID, "")
	require.NoError(t, err)
	require.Len(t, files, 2)
	require.Equal(t, "image.png", files[0].Name)
	require.Equal(t, "summary.md", files[1].Name)

	preview, err := svc.PreviewFile(context.Background(), entry.ID, "summary.md", 1024)
	require.NoError(t, err)
	require.Equal(t, "summary.md", preview.Name)
	require.Equal(t, "text/markdown; charset=utf-8", preview.ContentType)
	require.Contains(t, preview.Content, "# Summary")
}

func TestLocalServiceRejectsWorkspaceFilePathEscape(t *testing.T) {
	root := t.TempDir()
	svc := NewLocalService(root)
	entry, err := svc.Create(context.Background(), CreateInput{Name: "Reports"})
	require.NoError(t, err)

	_, err = svc.PreviewFile(context.Background(), entry.ID, "../secret.txt", 1024)
	require.ErrorIs(t, err, ErrPathEscape)
}

func TestLocalServicePreviewRejectsLargeText(t *testing.T) {
	root := t.TempDir()
	svc := NewLocalService(root)
	entry, err := svc.Create(context.Background(), CreateInput{Name: "Reports"})
	require.NoError(t, err)
	mustWriteFile(t, filepath.Join(root, entry.RelativePath, "large.txt"), strings.Repeat("a", 2048))

	_, err = svc.PreviewFile(context.Background(), entry.ID, "large.txt", 1024)
	require.ErrorIs(t, err, ErrFileTooLarge)
}
```

Add test helpers if they do not already exist:

```go
func mustWriteFile(t *testing.T, path string, data any) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	switch v := data.(type) {
	case string:
		require.NoError(t, os.WriteFile(path, []byte(v), 0o644))
	case []byte:
		require.NoError(t, os.WriteFile(path, v, 0o644))
	default:
		t.Fatalf("unsupported data type %T", data)
	}
}
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
go test ./internal/workspace -run "TestLocalService(ListFilesAndResolvePreview|RejectsWorkspaceFilePathEscape|PreviewRejectsLargeText)" -count=1
```

Expected: fail because `ListFiles`, `PreviewFile`, and `ErrFileTooLarge` do not exist.

- [ ] **Step 3: Extend workspace interfaces and types**

Add to `internal/types/interfaces/workspace.go`:

```go
ListFiles(ctx context.Context, workspaceID string, relDir string) ([]*workspace.FileEntry, error)
PreviewFile(ctx context.Context, workspaceID string, relPath string, maxBytes int64) (*workspace.FilePreview, error)
OpenFile(ctx context.Context, workspaceID string, relPath string) (*workspace.FileOpenResult, error)
```

Add to `internal/workspace/local.go`:

```go
var ErrFileTooLarge = errors.New("workspace file is too large to preview")

type FileEntry struct {
	Name         string    `json:"name"`
	RelativePath string   `json:"relative_path"`
	Kind         string   `json:"kind"`
	Size         int64    `json:"size"`
	ModifiedAt   time.Time `json:"modified_at"`
	IsDir        bool      `json:"is_dir"`
}

type FilePreview struct {
	Name         string `json:"name"`
	RelativePath string `json:"relative_path"`
	ContentType  string `json:"content_type"`
	Content      string `json:"content"`
	Size         int64  `json:"size"`
}

type FileOpenResult struct {
	Name         string
	RelativePath string
	AbsolutePath string
	ContentType  string
	Size         int64
}
```

- [ ] **Step 4: Implement safe file resolution**

Add a helper in `internal/workspace/local.go`:

```go
func (s *LocalService) resolveFilePath(ctx context.Context, workspaceID, relPath string) (*Entry, string, error) {
	entry, err := s.Resolve(ctx, workspaceID)
	if err != nil {
		return nil, "", err
	}
	cleanRel := filepath.Clean(strings.TrimSpace(relPath))
	if cleanRel == "." || cleanRel == "" {
		return entry, entry.RootPath, nil
	}
	if filepath.IsAbs(cleanRel) || strings.HasPrefix(cleanRel, ".."+string(os.PathSeparator)) || cleanRel == ".." {
		return nil, "", ErrPathEscape
	}
	abs := filepath.Join(entry.RootPath, cleanRel)
	rel, err := filepath.Rel(entry.RootPath, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil, "", ErrPathEscape
	}
	return entry, abs, nil
}
```

- [ ] **Step 5: Implement list/open/preview**

Implement:

```go
func (s *LocalService) ListFiles(ctx context.Context, workspaceID string, relDir string) ([]*FileEntry, error) {
	entry, dir, err := s.resolveFilePath(ctx, workspaceID, relDir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, ErrInvalidName
	}
	children, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]*FileEntry, 0, len(children))
	for _, child := range children {
		if strings.HasPrefix(child.Name(), ".") {
			continue
		}
		childInfo, err := child.Info()
		if err != nil {
			continue
		}
		abs := filepath.Join(dir, child.Name())
		rel, err := filepath.Rel(entry.RootPath, abs)
		if err != nil {
			continue
		}
		out = append(out, &FileEntry{
			Name: child.Name(), RelativePath: filepath.ToSlash(rel),
			Kind: detectFileKind(child.Name(), child.IsDir()),
			Size: childInfo.Size(), ModifiedAt: childInfo.ModTime(), IsDir: child.IsDir(),
		})
	}
	slices.SortFunc(out, func(a, b *FileEntry) int {
		if a.IsDir != b.IsDir {
			if a.IsDir { return -1 }
			return 1
		}
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return out, nil
}

func (s *LocalService) OpenFile(ctx context.Context, workspaceID string, relPath string) (*FileOpenResult, error) {
	_, abs, err := s.resolveFilePath(ctx, workspaceID, relPath)
	if err != nil { return nil, err }
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) { return nil, ErrNotFound }
		return nil, err
	}
	if info.IsDir() { return nil, ErrInvalidName }
	return &FileOpenResult{
		Name: filepath.Base(abs), RelativePath: filepath.ToSlash(filepath.Clean(relPath)),
		AbsolutePath: abs, ContentType: detectContentType(abs), Size: info.Size(),
	}, nil
}

func (s *LocalService) PreviewFile(ctx context.Context, workspaceID string, relPath string, maxBytes int64) (*FilePreview, error) {
	opened, err := s.OpenFile(ctx, workspaceID, relPath)
	if err != nil { return nil, err }
	if opened.Size > maxBytes { return nil, ErrFileTooLarge }
	if !isTextPreviewType(opened.RelativePath, opened.ContentType) { return nil, ErrUnsupportedPreview }
	data, err := os.ReadFile(opened.AbsolutePath)
	if err != nil { return nil, err }
	return &FilePreview{
		Name: opened.Name, RelativePath: opened.RelativePath,
		ContentType: opened.ContentType, Content: string(data), Size: opened.Size,
	}, nil
}
```

Also define `ErrUnsupportedPreview`, `detectFileKind`, `detectContentType`, and `isTextPreviewType` in the same package.

- [ ] **Step 6: Run tests and commit**

Run:

```powershell
go test ./internal/workspace -count=1
```

Expected: pass.

Commit:

```powershell
git add internal/types/interfaces/workspace.go internal/workspace/local.go internal/workspace/local_test.go
git commit -m "feat: add workspace file service"
```

## Task 2: Backend Workspace File HTTP API

**Files:**
- Modify: `internal/handler/workspace.go`
- Modify: `internal/handler/workspace_test.go`
- Modify: `internal/router/router.go`

- [ ] **Step 1: Write handler tests**

Add tests in `internal/handler/workspace_test.go`:

```go
func TestWorkspaceHandlerPreviewFileRejectsTraversal(t *testing.T) {
	h := NewWorkspaceHandler(newWorkspaceFileStub())
	r := gin.New()
	r.GET("/workspaces/:id/files/preview", h.PreviewFile)
	req := httptest.NewRequest(http.MethodGet, "/workspaces/ws-1/files/preview?path=../secret.txt", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "workspace_invalid_name")
}

func TestWorkspaceHandlerDownloadSetsAttachment(t *testing.T) {
	h := NewWorkspaceHandler(newWorkspaceFileStub())
	r := gin.New()
	r.GET("/workspaces/:id/files/download", h.DownloadFile)
	req := httptest.NewRequest(http.MethodGet, "/workspaces/ws-1/files/download?path=summary.md", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Disposition"), "attachment")
}
```

Use a stub implementing the extended `WorkspaceService`.

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
go test ./internal/handler -run "TestWorkspaceHandler(PreviewFileRejectsTraversal|DownloadSetsAttachment)" -count=1
```

Expected: fail because handler methods do not exist.

- [ ] **Step 3: Add handler methods**

Add in `internal/handler/workspace.go`:

```go
const workspacePreviewMaxBytes int64 = 1 << 20

func (h *WorkspaceHandler) ListFiles(c *gin.Context) {
	files, err := h.service.ListFiles(c.Request.Context(), strings.TrimSpace(c.Param("id")), c.Query("path"))
	if err != nil { writeWorkspaceError(c, err); return }
	c.JSON(http.StatusOK, gin.H{"files": files})
}

func (h *WorkspaceHandler) PreviewFile(c *gin.Context) {
	preview, err := h.service.PreviewFile(c.Request.Context(), strings.TrimSpace(c.Param("id")), c.Query("path"), workspacePreviewMaxBytes)
	if err != nil { writeWorkspaceError(c, err); return }
	c.JSON(http.StatusOK, preview)
}

func (h *WorkspaceHandler) RawFile(c *gin.Context) {
	opened, err := h.service.OpenFile(c.Request.Context(), strings.TrimSpace(c.Param("id")), c.Query("path"))
	if err != nil { writeWorkspaceError(c, err); return }
	c.Header("Content-Type", opened.ContentType)
	c.File(opened.AbsolutePath)
}

func (h *WorkspaceHandler) DownloadFile(c *gin.Context) {
	opened, err := h.service.OpenFile(c.Request.Context(), strings.TrimSpace(c.Param("id")), c.Query("path"))
	if err != nil { writeWorkspaceError(c, err); return }
	c.Header("Content-Type", opened.ContentType)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": opened.Name}))
	c.File(opened.AbsolutePath)
}
```

Extend `writeWorkspaceError` for:

```go
case errors.Is(err, workspace.ErrFileTooLarge):
	c.JSON(http.StatusRequestEntityTooLarge, gin.H{"code": "workspace_file_too_large", "message": "file is too large to preview"})
case errors.Is(err, workspace.ErrUnsupportedPreview):
	c.JSON(http.StatusUnsupportedMediaType, gin.H{"code": "workspace_preview_unsupported", "message": "file type is not supported for inline preview"})
```

- [ ] **Step 4: Register routes**

Modify `internal/router/router.go` under workspace routes:

```go
workspaces.GET("/:id/files", workspaceHandler.ListFiles)
workspaces.GET("/:id/files/preview", workspaceHandler.PreviewFile)
workspaces.GET("/:id/files/raw", workspaceHandler.RawFile)
workspaces.GET("/:id/files/download", workspaceHandler.DownloadFile)
```

- [ ] **Step 5: Run tests and commit**

Run:

```powershell
go test ./internal/handler -run WorkspaceHandler -count=1
go test ./internal/router -run Workspace -count=1
```

Expected: pass.

Commit:

```powershell
git add internal/handler/workspace.go internal/handler/workspace_test.go internal/router/router.go
git commit -m "feat: expose workspace file preview api"
```

## Task 3: Frontend File Preview Utilities And API

**Files:**
- Create: `frontend/src/utils/filePreview.ts`
- Modify: `frontend/src/api/workspace/index.ts`
- Test: `frontend/src/utils/filePreview.test.ts`

- [ ] **Step 1: Write utility tests**

Create `frontend/src/utils/filePreview.test.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { detectPreviewKind, fileInfoText, normalizeWorkspaceFileRef } from './filePreview';

describe('filePreview utilities', () => {
  it('detects supported preview kinds', () => {
    expect(detectPreviewKind('report.md', '')).toBe('markdown');
    expect(detectPreviewKind('notes.txt', '')).toBe('text');
    expect(detectPreviewKind('diagram.png', '')).toBe('image');
    expect(detectPreviewKind('contract.pdf', '')).toBe('pdf');
    expect(detectPreviewKind('sheet.xlsx', '')).toBe('spreadsheet');
  });

  it('normalizes workspace file refs with URLs', () => {
    const ref = normalizeWorkspaceFileRef('ws-1', {
      name: 'report.md', relative_path: 'docs/report.md', kind: 'markdown', size: 12,
    });
    expect(ref.source).toBe('workspace-file');
    expect(ref.workspaceId).toBe('ws-1');
    expect(ref.previewUrl).toContain('/api/v1/workspaces/ws-1/files/preview');
    expect(ref.downloadUrl).toContain('/api/v1/workspaces/ws-1/files/download');
  });

  it('copies relative file info only', () => {
    expect(fileInfoText({ source: 'workspace-file', name: 'a.md', path: 'a.md', relativePath: 'a.md' }))
      .toContain('Path: a.md');
  });
});
```

- [ ] **Step 2: Run test and verify failure**

Run:

```powershell
cd frontend
npm test -- filePreview.test.ts
```

Expected: fail because utility file does not exist.

- [ ] **Step 3: Extend workspace API**

Modify `frontend/src/api/workspace/index.ts`:

```ts
import { getApiBaseUrl } from '@/utils/api-base';

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

export async function listWorkspaceFiles(workspaceId: string, path = '') {
  const query = path ? `?path=${encodeURIComponent(path)}` : '';
  return get(`/api/v1/workspaces/${encodeURIComponent(workspaceId)}/files${query}`);
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
```

- [ ] **Step 4: Add utility implementation**

Create `frontend/src/utils/filePreview.ts` with:

```ts
import { workspaceFileDownloadUrl, workspaceFileRawUrl, type WorkspaceFileEntry } from '@/api/workspace';

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
  if (mimeType.startsWith('image/') || /\.(png|jpe?g|gif|webp|svg)$/i.test(lower)) return 'image';
  if (mimeType === 'application/pdf' || lower.endsWith('.pdf')) return 'pdf';
  if (/\.(xlsx?|csv)$/i.test(lower)) return 'spreadsheet';
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

export function fileInfoText(ref: PreviewFileRef) {
  return [
    `Name: ${ref.name}`,
    `Type: ${ref.kind || detectPreviewKind(ref.name, ref.mimeType)}`,
    ref.size != null ? `Size: ${ref.size} bytes` : null,
    `Path: ${ref.relativePath || ref.path}`,
    ref.workspaceId ? `Workspace: ${ref.workspaceId}` : null,
    ref.sessionId ? `Session: ${ref.sessionId}` : null,
  ].filter(Boolean).join('\n');
}
```

- [ ] **Step 5: Run tests and commit**

Run:

```powershell
cd frontend
npm test -- filePreview.test.ts
```

Expected: pass.

Commit:

```powershell
git add frontend/src/api/workspace/index.ts frontend/src/utils/filePreview.ts frontend/src/utils/filePreview.test.ts
git commit -m "feat: add file preview frontend model"
```

## Task 4: Global File Preview Drawer

**Files:**
- Create: `frontend/src/stores/filePreview.ts`
- Create: `frontend/src/components/file-preview/FilePreviewDrawer.vue`
- Create: `frontend/src/components/file-preview/FilePreviewActions.vue`
- Create: `frontend/src/components/file-preview/FilePreviewBody.vue`
- Create: preview components under `frontend/src/components/file-preview/previews/`
- Modify: `frontend/src/App.vue`

- [ ] **Step 1: Add store**

Create `frontend/src/stores/filePreview.ts`:

```ts
import { defineStore } from 'pinia';
import type { PreviewFileRef } from '@/utils/filePreview';

export const useFilePreviewStore = defineStore('filePreview', {
  state: () => ({
    visible: false,
    current: null as PreviewFileRef | null,
  }),
  actions: {
    open(file: PreviewFileRef) {
      this.current = file;
      this.visible = true;
    },
    close() {
      this.visible = false;
    },
    clear() {
      this.visible = false;
      this.current = null;
    },
  },
});
```

- [ ] **Step 2: Add action component**

Create `frontend/src/components/file-preview/FilePreviewActions.vue` with actions for preview, download, copy path, copy info, and open folder guidance. Use `navigator.clipboard.writeText` and `MessagePlugin.success/error`.

- [ ] **Step 3: Add preview body components**

Create:

```text
TextPreview.vue: calls previewWorkspaceFile(workspaceId, path) and renders markdown/text.
ImagePreview.vue: renders <img :src="file.rawUrl">.
PdfPreview.vue: renders <iframe :src="file.rawUrl">.
UnsupportedPreview.vue: shows metadata and unsupported message.
FilePreviewBody.vue: switches by kind and canInlinePreview(file).
```

For `TextPreview.vue`, use existing markdown renderer if easily importable; otherwise render `<pre>` for text and leave markdown styling to the next task. Do not block the first drawer on rich Markdown rendering.

- [ ] **Step 4: Add drawer**

Create `frontend/src/components/file-preview/FilePreviewDrawer.vue`:

```vue
<script setup lang="ts">
import { computed } from 'vue';
import { useFilePreviewStore } from '@/stores/filePreview';
import FilePreviewActions from './FilePreviewActions.vue';
import FilePreviewBody from './FilePreviewBody.vue';

const store = useFilePreviewStore();
const file = computed(() => store.current);
</script>

<template>
  <t-drawer
    :visible="store.visible"
    placement="right"
    size="620px"
    :footer="false"
    destroy-on-close
    @close="store.close"
  >
    <template #header>
      <div v-if="file" class="file-preview-header">
        <div class="file-preview-title">{{ file.name }}</div>
        <div class="file-preview-meta">
          <span>{{ file.kind || 'file' }}</span>
          <span v-if="file.size != null">{{ file.size }} bytes</span>
          <span>{{ file.source }}</span>
        </div>
      </div>
    </template>
    <div v-if="file" class="file-preview-drawer">
      <FilePreviewActions :file="file" />
      <FilePreviewBody :file="file" />
    </div>
  </t-drawer>
</template>
```

- [ ] **Step 5: Mount globally**

Modify `frontend/src/App.vue`:

```vue
<script setup lang="ts">
import FilePreviewDrawer from '@/components/file-preview/FilePreviewDrawer.vue'
</script>
<template>
  ...
  <FilePreviewDrawer />
</template>
```

- [ ] **Step 6: Run frontend type/build check and commit**

Run:

```powershell
cd frontend
npm run build
```

Expected: pass.

Commit:

```powershell
git add frontend/src/stores/filePreview.ts frontend/src/components/file-preview frontend/src/App.vue
git commit -m "feat: add global file preview drawer"
```

## Task 5: Workspace File Entry Point

**Files:**
- Modify or create a workspace UI surface after locating the current intended page.
- Prefer existing workspace selector/settings surface if no full workspace browser exists.

- [ ] **Step 1: Locate workspace page**

Run:

Create the first workspace file entry point in the chat workspace context:

- New component: `frontend/src/components/workspace-files/WorkspaceFilePanel.vue`
- Mount point: `frontend/src/views/chat/index.vue`

Expected: the chat page shows the currently bound workspace files without creating a new route or separate app section.

- [ ] **Step 2: Add file list API usage**

In the selected component, call:

```ts
const response = await listWorkspaceFiles(activeWorkspaceId.value, currentDir.value);
files.value = Array.isArray(response?.files) ? response.files : [];
```

Render rows with name, kind, size, and modified time. On file click:

```ts
filePreviewStore.open(normalizeWorkspaceFileRef(activeWorkspaceId.value, file));
```

- [ ] **Step 3: Verify manually and commit**

Run:

```powershell
cd frontend
npm run build
```

Expected: pass.

Commit:

```powershell
git add frontend/src/components/workspace-files/WorkspaceFilePanel.vue frontend/src/views/chat/index.vue
git commit -m "feat: open workspace files in preview drawer"
```

## Task 6: Chat Artifact Entry Point

**Files:**
- Modify likely file: `frontend/src/views/chat/components/AgentStreamDisplay.vue`
- Or modify specific tool-result component if artifacts are rendered there.

- [ ] **Step 1: Locate artifact rendering**

Run:

```powershell
rg -n "artifacts|artifact_detected|relative_path|preview_state|Artifact" frontend/src/views/chat frontend/src/components frontend/src/api
```

Expected: identify the component rendering executor artifacts or tool outputs containing artifact metadata.

- [ ] **Step 2: Add chat artifact normalization**

In the artifact rendering component:

```ts
import { useFilePreviewStore } from '@/stores/filePreview';
import { detectPreviewKind, type PreviewFileRef } from '@/utils/filePreview';

const filePreviewStore = useFilePreviewStore();

function normalizeChatArtifact(artifact: any): PreviewFileRef {
  const path = artifact.relative_path || artifact.path || artifact.name;
  const workspaceId = artifact.workspace_id || artifact.workspaceId;
  return {
    source: 'chat-artifact',
    name: artifact.name || path.split('/').pop() || path,
    path,
    relativePath: path,
    kind: artifact.kind || detectPreviewKind(path, artifact.content_type || ''),
    size: artifact.size,
    workspaceId,
    sessionId: artifact.session_id || artifact.sessionId,
    jobId: artifact.job_id || artifact.jobId,
    previewUrl: workspaceId ? `/api/v1/workspaces/${encodeURIComponent(workspaceId)}/files/preview?path=${encodeURIComponent(path)}` : undefined,
    rawUrl: workspaceId ? `/api/v1/workspaces/${encodeURIComponent(workspaceId)}/files/raw?path=${encodeURIComponent(path)}` : undefined,
    downloadUrl: workspaceId ? `/api/v1/workspaces/${encodeURIComponent(workspaceId)}/files/download?path=${encodeURIComponent(path)}` : undefined,
  };
}
```

Add click handler:

```ts
function openArtifactPreview(artifact: any) {
  filePreviewStore.open(normalizeChatArtifact(artifact));
}
```

- [ ] **Step 3: Add UI affordance**

Artifact rows/cards should include a button:

```vue
<t-button size="small" variant="text" @click.stop="openArtifactPreview(artifact)">
  <template #icon><t-icon name="file-search" /></template>
  预览
</t-button>
```

- [ ] **Step 4: Build and commit**

Run:

```powershell
cd frontend
npm run build
```

Expected: pass.

Commit:

```powershell
git add <artifact-rendering-files>
git commit -m "feat: open chat artifacts in preview drawer"
```

## Task 7: End-To-End Verification And Docker Deployment

**Files:**
- No source changes unless fixing defects discovered by verification.

- [ ] **Step 1: Run focused backend tests**

Run:

```powershell
go test ./internal/workspace ./internal/handler ./internal/router -count=1
```

Expected: pass.

- [ ] **Step 2: Run frontend build**

Run:

```powershell
cd frontend
npm run build
```

Expected: pass.

- [ ] **Step 3: Build Docker images**

Run from repo root:

```powershell
docker compose build app frontend
```

Expected: app and frontend images build successfully.

- [ ] **Step 4: Deploy locally**

Run:

```powershell
Copy-Item -LiteralPath 'E:\Xelora\WeKnora\frontend\dist' -Destination 'E:\Xelora\xelora-offline-20260714-011513\frontend-dist' -Recurse -Force
docker compose up -d app frontend
```

If the compose build context is the source repo rather than offline directory, run deployment from `E:\Xelora\WeKnora` first, then copy the images/tags into the offline compose naming scheme. Do not change database volumes during this task.

- [ ] **Step 5: Verify service health**

Run:

```powershell
docker compose ps
Invoke-RestMethod -Uri 'http://localhost:8080/health' -Method Get
```

Expected: `Xelora-app` healthy and health returns `{"status":"ok"}`.

- [ ] **Step 6: Verify API behavior**

Create sample files under an existing workspace directory, then run:

```powershell
$workspaceId = (Invoke-RestMethod -Uri 'http://localhost:8080/api/v1/workspaces' -Method Get)[0].id
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/workspaces/$workspaceId/files" -Method Get
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/workspaces/$workspaceId/files/preview?path=sample.md" -Method Get
Invoke-WebRequest -Uri "http://localhost:8080/api/v1/workspaces/$workspaceId/files/raw?path=sample.pdf" -Method Get
Invoke-WebRequest -Uri "http://localhost:8080/api/v1/workspaces/$workspaceId/files/preview?path=../secret.txt" -Method Get
```

Expected: list and preview succeed for valid files; traversal request returns `400`.

- [ ] **Step 7: Verify browser UI**

In the UI:

1. Open a workspace file list.
2. Click Markdown/text, image, and PDF files; each opens the right-side drawer.
3. Click an Office file; it shows unsupported preview with download/copy actions.
4. Open a chat artifact; it uses the same drawer.
5. Confirm download and copy actions work.

- [ ] **Step 8: Commit verification fixes or record completion**

If no code changes were needed:

```powershell
git status --short
```

Expected: clean except unrelated pre-existing files. If fixes were needed, commit them with a focused message.

## Self-Review Checklist

- Spec coverage:
  - Unified drawer: Tasks 3-6.
  - Chat artifact entry: Task 6.
  - Workspace file entry: Task 5.
  - Text/image/PDF preview: Tasks 2-4.
  - Unsupported Office state: Tasks 3-4.
  - Safe open menu: Task 4.
  - Backend path boundary checks: Tasks 1-2.
  - Docker deploy verification: Task 7.
- Red-flag scan: no unresolved markers or unspecified deferred steps are required to execute the plan.
- Type consistency: `PreviewFileRef`, `WorkspaceFileEntry`, and workspace API helper names are introduced before later tasks use them.
