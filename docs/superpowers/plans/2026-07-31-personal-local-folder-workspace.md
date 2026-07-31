# Personal Local Folder Workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** On Personal desktop, replace managed workspaces with single-root local folder binding so AI can list/read/write/create and (with confirm) delete/rename/move files inside a user-chosen directory.

**Architecture:** Keep `sessions.workspace_binding` and executor `WorkDir = root_path`. Personal bind accepts absolute `root_path`, derives `workspace_id = local:` + hash, sets `binding_kind=local_folder`. Recent folders live in client preferences. Destructive ops use propose → UI confirm → apply with short-lived tokens. Docker/Enterprise managed flow unchanged.

**Tech Stack:** Go (Gin, existing workspace package), Wails v2 dialogs, Vue 3 + Pinia + TDesign, existing session binding + agent approval patterns.

**Spec:** `docs/superpowers/specs/2026-07-31-personal-local-folder-workspace-design.md`

---

## File map

| File | Responsibility |
|---|---|
| `internal/types/session.go` | `RootPath` on bind input; `BindingKind` on binding |
| `internal/workspace/local_id.go` | Normalize path + `local:` workspace id hash |
| `internal/workspace/local_folder.go` | Bind/validate absolute local folder (Personal) |
| `internal/workspace/mutation.go` | In-memory pending mutations + apply under root |
| `internal/application/service/session.go` | Personal `normalizeWorkspaceBinding` path |
| `internal/handler/session/handler.go` | Pass `RootPath` into session binding |
| `internal/handler/workspace.go` | Mutation propose/apply routes; disable Create on Personal |
| `internal/router/router.go` | Register mutation routes |
| `internal/agent/tools/workspace_mutate.go` | Agent tool for delete/rename/move (waits on confirm) |
| `cmd/desktop/app.go` | `SelectDirectory()` Wails dialog |
| `frontend/src/wailsjs/go/main/App.*` | Bindings for SelectDirectory (or dynamic `window.go`) |
| `frontend/src/utils/recentLocalWorkspaces.ts` | Recent list preference helpers |
| `frontend/src/components/WorkspaceSelector.vue` | Personal: Open Folder + recent; hide create |
| `frontend/src/views/creatChat/creatChat.vue` | Send `root_path` on create |
| `frontend/src/components/workspace-files/WorkspaceMutationConfirm.vue` | Confirm dialog host |
| `frontend/src/i18n/locales/zh-CN.ts`, `en-US.ts` | Strings |

---

### Task 1: Local folder id + bind helpers

**Files:**
- Create: `internal/workspace/local_id.go`
- Create: `internal/workspace/local_id_test.go`
- Create: `internal/workspace/local_folder.go`
- Create: `internal/workspace/local_folder_test.go`

- [ ] **Step 1: Write failing id tests**

```go
package workspace_test

func TestLocalWorkspaceID_Stable(t *testing.T) {
	a := workspace.LocalWorkspaceID(`D:\proj\Foo`)
	b := workspace.LocalWorkspaceID(`D:\proj\Foo`)
	require.True(t, strings.HasPrefix(a, "local:"))
	require.Equal(t, a, b)
}

func TestLocalWorkspaceID_Normalizes(t *testing.T) {
	// Same directory via cleaned path → same id (platform-specific clean)
	id1 := workspace.LocalWorkspaceID(filepath.Clean(`D:\proj\Foo`))
	id2 := workspace.LocalWorkspaceID(filepath.Clean(`D:\proj\Foo\`))
	require.Equal(t, id1, id2)
}
```

- [ ] **Step 2: Run tests — expect FAIL**

```bash
go test ./internal/workspace/ -run "LocalWorkspaceID" -count=1
```

Expected: FAIL — undefined `LocalWorkspaceID`

- [ ] **Step 3: Implement helpers**

```go
// local_id.go
func NormalizeLocalRoot(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", ErrInvalidName
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func LocalWorkspaceID(normalizedAbs string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(normalizedAbs))))
	return "local:" + hex.EncodeToString(sum[:16])
}

func LocalWorkspaceName(normalizedAbs string) string {
	base := filepath.Base(normalizedAbs)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return normalizedAbs
	}
	return base
}
```

```go
// local_folder.go
type LocalFolderEntry struct {
	ID       string
	Name     string
	RootPath string
	Status   string // StatusAvailable / StatusMissing / StatusAccessDenied
}

func BindLocalFolder(rootPath string) (*LocalFolderEntry, error) {
	norm, err := NormalizeLocalRoot(rootPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(norm)
	if err != nil {
		if os.IsNotExist(err) {
			return &LocalFolderEntry{ID: LocalWorkspaceID(norm), Name: LocalWorkspaceName(norm), RootPath: norm, Status: StatusMissing}, ErrNotFound
		}
		if os.IsPermission(err) {
			return &LocalFolderEntry{ID: LocalWorkspaceID(norm), Name: LocalWorkspaceName(norm), RootPath: norm, Status: StatusAccessDenied}, ErrAccessDenied
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: not a directory", ErrInvalidName)
	}
	return &LocalFolderEntry{
		ID: LocalWorkspaceID(norm), Name: LocalWorkspaceName(norm),
		RootPath: norm, Status: StatusAvailable,
	}, nil
}
```

- [ ] **Step 4: Tests for BindLocalFolder with temp dir**

```go
func TestBindLocalFolder_OK(t *testing.T) {
	dir := t.TempDir()
	ent, err := workspace.BindLocalFolder(dir)
	require.NoError(t, err)
	require.Equal(t, workspace.StatusAvailable, ent.Status)
	require.True(t, strings.HasPrefix(ent.ID, "local:"))
}
```

- [ ] **Step 5: Run tests — expect PASS**

```bash
go test ./internal/workspace/ -run "LocalWorkspaceID|BindLocalFolder" -count=1
```

- [ ] **Step 6: Commit**

```bash
git add internal/workspace/local_id.go internal/workspace/local_id_test.go \
  internal/workspace/local_folder.go internal/workspace/local_folder_test.go
git commit -m "feat: local folder workspace id and bind helpers"
```

---

### Task 2: Session binding types + Personal normalize

**Files:**
- Modify: `internal/types/session.go`
- Modify: `internal/application/service/session.go`
- Modify: `internal/handler/session/handler.go`
- Create: `internal/application/service/session_local_folder_binding_test.go`

- [ ] **Step 1: Extend types**

```go
type SessionWorkspaceBindingKind string

const (
	SessionWorkspaceBindingKindManaged     SessionWorkspaceBindingKind = "managed"
	SessionWorkspaceBindingKindLocalFolder SessionWorkspaceBindingKind = "local_folder"
)

type SessionWorkspaceBindingInput struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	RootPath    string `json:"root_path,omitempty"` // Personal only
}

type SessionWorkspaceBinding struct {
	// existing fields...
	BindingKind SessionWorkspaceBindingKind `json:"binding_kind,omitempty"`
}
```

- [ ] **Step 2: Write failing normalize test (Personal)**

```go
func TestNormalizeWorkspaceBinding_PersonalLocalFolder(t *testing.T) {
	prev := handlerpkg.Edition
	handlerpkg.Edition = "personal"
	t.Cleanup(func() { handlerpkg.Edition = prev })

	dir := t.TempDir()
	svc := &sessionService{workspaceService: nil} // local path must not need registry
	// Prefer injecting edition check via workspace helper or sessionService flag —
	// use the same Edition package var the handler uses.

	out, err := svc.normalizeWorkspaceBinding(context.Background(), "user-1", &types.SessionWorkspaceBinding{
		RootPath: dir,
	})
	require.NoError(t, err)
	require.Equal(t, types.SessionWorkspaceBindingKindLocalFolder, out.BindingKind)
	require.True(t, strings.HasPrefix(out.WorkspaceID, "local:"))
	require.Equal(t, types.SessionWorkspaceBindingStatusBound, out.Status)
}
```

Note: if `Edition` lives in `internal/handler`, avoid import cycles — either move `IsPersonalEdition()` to `internal/runtimeedition` / check env `XELORA` build ldflag via existing `handler.Edition` from a tiny `internal/edition` package, **or** pass a bool/`ResolveMode` on `sessionService`. Preferred in this codebase: read `os.Getenv` is wrong; use existing `handler.Edition` only from handlers. For service tests, add:

```go
// in sessionService
personalLocalFolders bool // set true from container when Edition==personal
```

Wire in `internal/container/container.go` when building session service for Personal.

- [ ] **Step 3: Implement normalize branch**

```go
func (s *sessionService) normalizeWorkspaceBinding(...) (*types.SessionWorkspaceBinding, error) {
	if binding == nil {
		return nil, nil
	}
	now := time.Now()
	rootPath := strings.TrimSpace(binding.RootPath)
	workspaceID := strings.TrimSpace(binding.WorkspaceID)

	if s.personalLocalFolders {
		if rootPath == "" {
			// Legacy managed id without path → invalid (migration)
			if workspaceID == "" {
				return nil, nil
			}
			return &types.SessionWorkspaceBinding{
				WorkspaceID: workspaceID, Status: types.SessionWorkspaceBindingStatusInvalid,
				ValidationMessage: "请打开本机文件夹重新绑定工作区",
				BoundByUserID: userID, BoundAt: cloneTimePtr(binding.BoundAt, &now), LastValidatedAt: &now,
				BindingKind: types.SessionWorkspaceBindingKindLocalFolder,
			}, fmt.Errorf("binding_invalid: local folder root_path required")
		}
		ent, err := workspace.BindLocalFolder(rootPath)
		if err != nil && ent == nil {
			return nil, err
		}
		if workspaceID != "" && ent != nil && workspaceID != ent.ID {
			return nil, fmt.Errorf("binding_invalid: workspace_id does not match root_path")
		}
		status := types.SessionWorkspaceBindingStatusBound
		if ent.Status == workspace.StatusMissing {
			status = types.SessionWorkspaceBindingStatusInvalid
		}
		if ent.Status == workspace.StatusAccessDenied {
			status = types.SessionWorkspaceBindingStatusAccessDenied
		}
		out := &types.SessionWorkspaceBinding{
			WorkspaceID: ent.ID, WorkspaceName: ent.Name, RootPath: ent.RootPath,
			Status: status, BindingKind: types.SessionWorkspaceBindingKindLocalFolder,
			BoundByUserID: userID, BoundAt: cloneTimePtr(binding.BoundAt, &now), LastValidatedAt: &now,
		}
		if status != types.SessionWorkspaceBindingStatusBound {
			return out, fmt.Errorf("binding_invalid: %s", ent.Status)
		}
		return out, nil
	}

	// existing managed Resolve(workspaceID) path...
}
```

- [ ] **Step 4: Handler passes RootPath**

```go
createdSession.WorkspaceBinding = &types.SessionWorkspaceBinding{
	WorkspaceID: request.WorkspaceBinding.WorkspaceID,
	RootPath:    request.WorkspaceBinding.RootPath,
}
// same for UpdateSession
```

- [ ] **Step 5: Extend `isWorkspaceBindingValidationError` to include `binding_invalid`**

- [ ] **Step 6: Run tests**

```bash
go test ./internal/application/service/ -run "NormalizeWorkspaceBinding_Personal" -count=1
go test ./internal/handler/session/ -run "WorkspaceBinding" -count=1
```

- [ ] **Step 7: Commit**

```bash
git add internal/types/session.go internal/application/service/session.go \
  internal/application/service/session_local_folder_binding_test.go \
  internal/handler/session/handler.go internal/container/container.go
git commit -m "feat: Personal session bind via local folder root_path"
```

---

### Task 3: Disable managed Create on Personal + file APIs use bound root

**Files:**
- Modify: `internal/handler/workspace.go`
- Modify: `internal/handler/workspace_test.go` (or new test)
- Verify: list/preview already resolve via workspace id — for Personal local ids, add `Resolve` path

- [ ] **Step 1: LocalRegistry.Resolve for `local:` ids OR dedicated Personal resolver**

Prefer extending workspace service interface:

```go
// On Personal, WorkspaceService.Resolve("local:...") looks up is wrong without registry.
// Instead: session binding already stores RootPath; file handlers should prefer
// session-bound root when listing by workspace_id matching binding.
```

Concrete approach for file routes that take `:id`:

1. Add `LocalRegistry.ResolveLocal(id, rootHint string)` — **not** used.
2. Better: maintain a process-local map filled on successful bind:

```go
// internal/workspace/local_folder_index.go
var localIndex sync.Map // id -> rootPath

func RememberLocalFolder(id, root string) { localIndex.Store(id, root) }
func LookupLocalFolder(id string) (string, bool) { ... }
```

Call `RememberLocalFolder` from successful Personal normalize. `Resolve` checks prefix `local:` then Lookup + Stat.

- [ ] **Step 2: Create on Personal returns 400**

```go
if handler.Edition == "personal" {
	c.JSON(http.StatusBadRequest, gin.H{"error": "managed_workspace_disabled", "message": "Personal uses local folders"})
	return
}
```

- [ ] **Step 3: Tests + commit**

```bash
go test ./internal/workspace/ ./internal/handler/ -run "LocalFolder|CreateWorkspace" -count=1
git commit -m "feat: Personal local folder resolve index; disable managed create"
```

---

### Task 4: Destructive mutations propose / apply

**Files:**
- Create: `internal/workspace/mutation.go`
- Create: `internal/workspace/mutation_test.go`
- Modify: `internal/handler/workspace.go`
- Modify: `internal/router/router.go`

- [ ] **Step 1: Failing tests for confine + apply**

```go
func TestApplyMutation_Delete(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "a.txt")
	require.NoError(t, os.WriteFile(target, []byte("x"), 0o644))
	store := workspace.NewMutationStore(time.Minute)
	p, err := store.Propose(workspace.MutationProposal{
		SessionID: "s1", WorkspaceID: "local:x", RootPath: root,
		Op: "delete", FromRel: "a.txt",
	})
	require.NoError(t, err)
	require.NoError(t, store.Apply(p.ID, p.Token))
	_, err = os.Stat(target)
	require.True(t, os.IsNotExist(err))
}

func TestApplyMutation_RejectsEscape(t *testing.T) {
	root := t.TempDir()
	store := workspace.NewMutationStore(time.Minute)
	_, err := store.Propose(workspace.MutationProposal{
		SessionID: "s1", WorkspaceID: "local:x", RootPath: root,
		Op: "delete", FromRel: "../outside.txt",
	})
	require.ErrorIs(t, err, workspace.ErrPathEscape)
}
```

- [ ] **Step 2: Implement MutationStore**

```go
type MutationProposal struct {
	ID, Token, SessionID, WorkspaceID, RootPath string
	Op string // delete | rename | move
	FromRel, ToRel string
	ExpiresAt time.Time
}

func (s *MutationStore) Propose(p MutationProposal) (*MutationProposal, error)
func (s *MutationStore) Apply(id, token string) error
func (s *MutationStore) Reject(id, token string) error
```

Path join must use existing confine helper from `local.go` (reuse `resolveUnderRoot` if present; else extract).

- [ ] **Step 3: HTTP routes**

```
POST /api/v1/workspaces/mutations/propose
POST /api/v1/workspaces/mutations/:id/apply
POST /api/v1/workspaces/mutations/:id/reject
```

Body includes session_id + op + paths; server loads session binding, verifies root, proposes.

- [ ] **Step 4: Tests + commit**

```bash
go test ./internal/workspace/ -run "ApplyMutation" -count=1
git commit -m "feat: workspace mutation propose/apply with path confinement"
```

---

### Task 5: Agent tool `workspace_mutate`

**Files:**
- Create: `internal/agent/tools/workspace_mutate.go`
- Create: `internal/agent/tools/workspace_mutate_test.go`
- Wire tool registration where other workspace tools register

- [ ] **Step 1: Tool behavior**

- Args: `op` (`delete`|`rename`|`move`), `from`, `to` (relative)
- Creates propose via MutationStore
- Emits event / returns pending payload for UI (mirror MCP approval event shape if chat already listens — prefer SSE/`tool_approval`-like payload: `workspace_mutation_required`)
- Blocks until apply/reject/timeout (reuse pattern from `approval.Gate.RequestAndWait` **or** simpler channel wait in Personal-only tool)

Minimal Personal approach:

```go
// Tool returns immediately with:
// {"status":"pending_confirmation","proposal_id":"...","token":"...","op":"delete","from":"a.txt"}
// Frontend confirm calls apply; next agent turn sees file gone.
// OR tool waits on in-process Wait(proposalID) like MCP gate.
```

**Locked for this plan:** tool **waits** (like MCP approval) so one agent turn completes after user confirms. Emit `EventWorkspaceMutationRequired` with proposal fields; frontend posts apply/reject; tool unblocks.

- [ ] **Step 2: Unit test wait approve/reject with fake bus**

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: workspace_mutate agent tool with confirm wait"
```

---

### Task 6: Wails SelectDirectory

**Files:**
- Modify: `cmd/desktop/app.go`
- Modify: `frontend/src/wailsjs/go/main/App.js` and `App.d.ts` **or** call via `window.go.main.App.SelectDirectory` dynamically (repo already uses `-skipbindings` — prefer dynamic invoke + hand-written d.ts snippet)

- [ ] **Step 1: Implement**

```go
func (a *App) SelectDirectory() (string, error) {
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择工作区文件夹",
	})
}
```

- [ ] **Step 2: Manual smoke from Personal console / temporary button**

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: Wails SelectDirectory for local workspace picker"
```

---

### Task 7: Frontend recent list + Personal WorkspaceSelector

**Files:**
- Create: `frontend/src/utils/recentLocalWorkspaces.ts`
- Create: `frontend/src/utils/recentLocalWorkspaces.test.ts`
- Modify: `frontend/src/components/WorkspaceSelector.vue`
- Modify: `frontend/src/views/creatChat/creatChat.vue`
- Modify: `frontend/src/api/chat/index.ts` (input type `root_path`)
- Modify: i18n zh-CN / en-US

- [ ] **Step 1: Recent helpers**

```ts
export type RecentLocalWorkspace = {
  workspace_id: string
  workspace_name: string
  root_path: string
  last_used_at: string
}
const KEY = 'xelora_recent_local_workspaces'
const MAX = 20

export function loadRecentLocalWorkspaces(): RecentLocalWorkspace[]
export function rememberRecentLocalWorkspace(entry: Omit<RecentLocalWorkspace, 'last_used_at'>): void
export function dismissRecentLocalWorkspace(workspace_id: string): void
```

- [ ] **Step 2: Personal selector UX**

When `authStore.isPersonalMode`:

- Hide create-managed button
- Button「打开文件夹…」→ `SelectDirectory()` → set `modelValue` to path object
- Change v-model contract: emit either workspace id (managed) **or** for Personal emit `root_path` via expanded model:

```ts
// Prefer structured model for Personal:
export type WorkspaceSelection =
  | { kind: 'managed'; workspace_id: string }
  | { kind: 'local_folder'; root_path: string; workspace_id?: string }
```

Update `creatChat.vue`:

```ts
sessionData.workspace_binding = selection.kind === 'local_folder'
  ? { root_path: selection.root_path, workspace_id: selection.workspace_id }
  : { workspace_id: selection.workspace_id }
```

Show recent list as options; selecting one sends that `root_path`.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: Personal Open Folder workspace selector and recent list"
```

---

### Task 8: Mutation confirm UI

**Files:**
- Create: `frontend/src/components/workspace-files/WorkspaceMutationConfirm.vue`
- Wire into chat shell (same place MCP tool approval UI mounts — search `tool_approval` / `ToolApproval`)

- [ ] **Step 1: Find existing approval UI and mirror**

Search: `ToolApproval`, `approval_required`, `mcp_approval` in `frontend/src`.

- [ ] **Step 2: Dialog**

Show op + from/to; Confirm → `POST /workspaces/mutations/:id/apply`; Cancel → reject.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: confirm UI for workspace delete/rename/move"
```

---

### Task 9: End-to-end verification (Personal)

**Files:** none required beyond fixes

- [ ] **Step 1: Unit suite**

```bash
go test ./internal/workspace/ ./internal/application/service/ ./internal/handler/session/ ./internal/agent/tools/ -count=1
```

- [ ] **Step 2: Rebuild Personal**

```powershell
Get-Process -Name "Xelora Personal" -ErrorAction SilentlyContinue | Stop-Process -Force
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-personal.ps1
```

- [ ] **Step 3: Manual checklist**

1. Open Folder → create chat → binding shows folder name  
2. Ask AI to create a file → appears on disk  
3. Ask AI to delete/rename → confirm dialog → disk updates only after confirm  
4. Reject mutation → disk unchanged  
5. Delete folder on disk → rebind prompt / invalid binding  
6. No「创建托管工作区」入口  

- [ ] **Step 4: Commit fixes if any**

```bash
git commit -m "fix: Personal local folder workspace verification follow-ups"
```

---

## Spec coverage

| Spec item | Task |
|---|---|
| Personal-only local folder | 2, 3, 7 |
| Replace managed on Personal | 3, 7 |
| Single root | 2, 7 |
| Recent list client-side | 7 |
| Bind always sends root_path | 2, 7 |
| Read/write/create immediate | existing + 3 Resolve |
| Delete/rename/move confirm | 4, 5, 8 |
| Path confinement | 1, 4 |
| Migration invalid managed | 2 |
| Wails dialog | 6 |
| Non-Personal unchanged | 2 (branch), 3 Create guard |

## Out of scope (do not implement)

Multi-root, terminal, sensitive denylist, Docker arbitrary paths, remote reverse tunnel.
