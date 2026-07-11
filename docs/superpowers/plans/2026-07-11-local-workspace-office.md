# Local Workspace And Office Editing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bind new conversations to a persistent administrator-approved Windows folder and make all file and Office tools create and repeatedly edit real files there.

**Architecture:** Mount one configured host root into `/workspaces`, expose a small Xelora-owned workspace registry/API, and resolve session bindings server-side. The executor runs each file-producing skill from the bound directory, stages request files under a hidden per-job directory, blocks unbound writes, and registers only real workspace outputs as artifacts.

**Tech Stack:** Go 1.26, Gin, Vue 3.5, TypeScript 6, Pinia, Docker Compose, Python 3, OfficeCLI

## Global Constraints

- Mount only `XELORA_WORKSPACE_HOST_ROOT`; never mount a whole drive implicitly.
- Browser clients submit `workspace_id`, never host or container paths.
- Existing chat remains available without a workspace, but file-producing jobs return `workspace_required` and never fall back to skill-private directories.
- Keep Controlled Docker as the first verified provider and preserve provider-independent session and artifact contracts.
- Keep OfficeCLI as the only Office provider in this implementation; do not add Python Office SDK dependencies yet.
- Preserve unrelated changes in the dirty worktree and commit only files belonging to each task.

---

### Task 1: Host Workspace Registry

**Files:**
- Create: `internal/workspace/local.go`
- Create: `internal/workspace/local_test.go`

**Interfaces:**
- Produces: `workspace.LocalRegistry`, `workspace.Entry`, `workspace.CreateInput`, `List(context.Context)`, `Create(context.Context, CreateInput)`, and `Resolve(context.Context, string)`.
- Consumes: `XELORA_WORKSPACE_CONTAINER_ROOT`; tests construct the registry with `NewLocalRegistry(root string)`.

- [ ] **Step 1: Write failing registry tests**

```go
func TestLocalRegistryCreateListResolve(t *testing.T) {
    root := t.TempDir()
    registry := NewLocalRegistry(root)
    ctx := workspaceTestContext(7, "user-1")
    created, err := registry.Create(ctx, CreateInput{Name: "Quarterly Review"})
    require.NoError(t, err)
    require.Equal(t, "Quarterly Review", created.Name)
    require.DirExists(t, filepath.Join(root, created.RelativePath))

    listed, err := registry.List(ctx)
    require.NoError(t, err)
    require.Len(t, listed, 1)

    resolved, err := registry.Resolve(ctx, created.ID)
    require.NoError(t, err)
    require.Equal(t, filepath.Join(root, created.RelativePath), resolved.RootPath)
}

func TestLocalRegistryRejectsEscapesAndSymlinks(t *testing.T) {
    root := t.TempDir()
    registry := NewLocalRegistry(root)
    _, err := registry.Create(context.Background(), CreateInput{Name: "../outside"})
    require.ErrorIs(t, err, ErrPathEscape)
}
```

- [ ] **Step 2: Run tests and verify failure**

Run: `go test ./internal/workspace -run TestLocalRegistry -v`

Expected: FAIL because `NewLocalRegistry`, `CreateInput`, and registry methods do not exist.

- [ ] **Step 3: Implement the registry with an atomic metadata file**

```go
type Entry struct {
    ID           string `json:"id"`
    Name         string `json:"name"`
    RelativePath string `json:"relative_path"`
    RootPath     string `json:"root_path,omitempty"`
    Status       string `json:"status"`
    TenantID     uint64 `json:"-"`
    UserID       string `json:"-"`
}

type CreateInput struct {
    Name string `json:"name"`
}

type LocalRegistry struct {
    root string
    mu   sync.Mutex
}

func NewLocalRegistry(root string) *LocalRegistry {
    return &LocalRegistry{root: filepath.Clean(strings.TrimSpace(root))}
}

func (r *LocalRegistry) Create(ctx context.Context, input CreateInput) (*Entry, error) {
    r.mu.Lock()
    defer r.mu.Unlock()
    name, err := validateDirectoryName(input.Name)
    if err != nil { return nil, err }
    if err := r.ensureRoot(); err != nil { return nil, err }
    target := filepath.Join(r.root, name)
    if !isWithinRoot(r.root, target) { return nil, ErrPathEscape }
    if err := os.Mkdir(target, 0o755); err != nil && !errors.Is(err, os.ErrExist) { return nil, err }
    tenantID := types.MustTenantIDFromContext(ctx)
    userID, _ := types.UserIDFromContext(ctx)
    entry := Entry{ID: uuid.NewString(), Name: name, RelativePath: name, Status: "available", TenantID: tenantID, UserID: userID}
    if err := r.appendEntry(entry); err != nil { return nil, err }
    entry.RootPath = target
    return &entry, nil
}
```

`appendEntry` reads `.xelora-workspaces.json`, rejects duplicate names within the active tenant/user scope, writes a sibling temporary file with mode `0600`, and atomically renames it. Registry metadata persists `tenant_id` and `user_id` through a private disk DTO while the public `Entry` JSON omits both. `Resolve` must reject an ID owned by another tenant or user, join `RelativePath` to `root`, evaluate existing symlinks with `filepath.EvalSymlinks`, check `filepath.Rel` does not begin with `..`, and verify read/write access by opening the directory and creating/removing a temporary probe file. `List` returns only entries visible to the current tenant/user, sorted by name, and derives current status without exposing `RootPath` to HTTP callers.

- [ ] **Step 4: Run registry tests**

Run: `go test ./internal/workspace -v`

Expected: PASS, including missing root, duplicate name, traversal, missing directory, and symlink escape cases.

- [ ] **Step 5: Commit**

```powershell
git add internal/workspace/local.go internal/workspace/local_test.go
git commit -m "feat: add local workspace registry"
```

### Task 2: Workspace HTTP API And Dependency Wiring

**Files:**
- Create: `internal/types/interfaces/workspace.go`
- Create: `internal/handler/workspace.go`
- Create: `internal/handler/workspace_test.go`
- Modify: `internal/container/container.go`
- Modify: `internal/router/router.go`

**Interfaces:**
- Consumes: `workspace.LocalRegistry` from Task 1.
- Produces: authenticated `GET /api/v1/workspaces`, `POST /api/v1/workspaces`, and `GET /api/v1/workspaces/:id`.

- [ ] **Step 1: Write handler tests**

```go
type workspaceServiceStub struct {
    entries []*workspace.Entry
    created *workspace.Entry
}

func TestWorkspaceHandlerList(t *testing.T) {
    stub := &workspaceServiceStub{entries: []*workspace.Entry{{ID: "ws-1", Name: "Reports", RelativePath: "Reports", Status: "available"}}}
    handler := NewWorkspaceHandler(stub)
    recorder, c := testContext(http.MethodGet, "/api/v1/workspaces", nil)
    handler.List(c)
    require.Equal(t, http.StatusOK, recorder.Code)
    require.NotContains(t, recorder.Body.String(), "root_path")
}
```

- [ ] **Step 2: Run tests and verify failure**

Run: `go test ./internal/handler -run WorkspaceHandler -v`

Expected: FAIL because the handler and interface do not exist.

- [ ] **Step 3: Add the interface and handler**

```go
type WorkspaceService interface {
    List(ctx context.Context) ([]*workspace.Entry, error)
    Create(ctx context.Context, input workspace.CreateInput) (*workspace.Entry, error)
    Resolve(ctx context.Context, id string) (*workspace.Entry, error)
}

type WorkspaceHandler struct { service interfaces.WorkspaceService }

func (h *WorkspaceHandler) Create(c *gin.Context) {
    var input workspace.CreateInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "message": err.Error()})
        return
    }
    entry, err := h.service.Create(c.Request.Context(), input)
    if err != nil { writeWorkspaceError(c, err); return }
    entry.RootPath = ""
    c.JSON(http.StatusCreated, entry)
}
```

Register the service through Dig with a provider that returns the interface, provide `handler.NewWorkspaceHandler`, add it to `RouterParams`, and register routes under the existing authenticated `/api/v1` group with `Viewer()` guards.

```go
func newWorkspaceService() interfaces.WorkspaceService {
    root := strings.TrimSpace(os.Getenv("XELORA_WORKSPACE_CONTAINER_ROOT"))
    if root == "" { root = "/workspaces" }
    return workspace.NewLocalRegistry(root)
}
```

- [ ] **Step 4: Run handler and router tests**

Run: `go test ./internal/handler ./internal/router ./internal/container`

Expected: PASS; list/create responses never contain host or canonical container paths.

- [ ] **Step 5: Commit**

```powershell
git add internal/types/interfaces/workspace.go internal/handler/workspace.go internal/handler/workspace_test.go internal/container/container.go internal/router/router.go
git commit -m "feat: expose local workspace API"
```

### Task 3: Authoritative Session Workspace Binding

**Files:**
- Modify: `internal/application/service/session.go`
- Modify: `internal/application/service/session_user_scope_test.go`
- Modify: `internal/types/session.go`
- Modify: `internal/handler/session/handler.go`
- Modify: `internal/handler/session/types.go`

**Interfaces:**
- Consumes: `interfaces.WorkspaceService.Resolve(ctx, workspaceID)`.
- Produces: session bindings whose name, canonical root, status, and audit fields are server-derived.

- [ ] **Step 1: Add failing session binding tests**

```go
func TestCreateSessionResolvesWorkspaceIDServerSide(t *testing.T) {
    ws := &workspaceServiceStub{resolved: &workspace.Entry{ID: "ws-1", Name: "Reports", RootPath: "/workspaces/Reports", Status: "available"}}
    service := newSessionServiceForTest(t, withWorkspaceService(ws))
    created, err := service.CreateSession(ctx, &types.Session{TenantID: 7, UserID: "user-1", WorkspaceBinding: &types.SessionWorkspaceBinding{WorkspaceID: "ws-1", RootPath: "/forged"}})
    require.NoError(t, err)
    require.Equal(t, "/workspaces/Reports", created.WorkspaceBinding.RootPath)
}

func TestCreateSessionRejectsUnknownWorkspace(t *testing.T) {
    ws := &workspaceServiceStub{resolveErr: workspace.ErrNotFound}
    service := newSessionServiceForTest(t, withWorkspaceService(ws))
    _, err := service.CreateSession(ctx, &types.Session{TenantID: 7, WorkspaceBinding: &types.SessionWorkspaceBinding{WorkspaceID: "missing"}})
    require.ErrorContains(t, err, "workspace_not_found")
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run: `go test ./internal/application/service -run Workspace -v`

Expected: FAIL because the current normalizer accepts only `tenant:<id>` and derives `/data/files/session-workspaces/...`.

- [ ] **Step 3: Inject `WorkspaceService` and replace tenant-derived normalization**

```go
func (s *sessionService) normalizeWorkspaceBinding(ctx context.Context, userID string, binding *types.SessionWorkspaceBinding) (*types.SessionWorkspaceBinding, error) {
    if binding == nil || strings.TrimSpace(binding.WorkspaceID) == "" { return nil, nil }
    entry, err := s.workspaceService.Resolve(ctx, strings.TrimSpace(binding.WorkspaceID))
    if err != nil { return blockedBinding(binding.WorkspaceID, userID, err) }
    now := time.Now()
    return &types.SessionWorkspaceBinding{
        WorkspaceID: entry.ID, WorkspaceName: entry.Name, RootPath: entry.RootPath,
        Status: types.SessionWorkspaceBindingStatusBound,
        BoundAt: cloneTimePtr(binding.BoundAt, &now), BoundByUserID: userID,
        LastValidatedAt: &now,
    }, nil
}
```

Remove `RootPath` and `WorkspaceName` from `SessionWorkspaceBindingInput` so new clients cannot forge them. Keep response fields on `SessionWorkspaceBinding` for hydration. Preserve explicit `null` clearing and omitted-field preservation from existing handler tests. Legacy `tenant:*` values resolve to `invalid` with a selection-required message.

- [ ] **Step 4: Run session tests**

Run: `go test ./internal/application/service ./internal/handler/session ./internal/application/repository`

Expected: PASS for server-derived paths, unknown IDs, omitted binding, explicit null, and legacy sessions.

- [ ] **Step 5: Commit**

```powershell
git add internal/application/service/session.go internal/application/service/session_user_scope_test.go internal/types/session.go internal/handler/session/handler.go internal/handler/session/types.go
git commit -m "feat: resolve session workspaces server side"
```

### Task 4: Executor Workspace Enforcement And Request Staging

**Files:**
- Modify: `internal/executor/gateway.go`
- Modify: `internal/executor/gateway_test.go`
- Modify: `internal/executor/types.go`
- Modify: `internal/agent/skills/manager.go`

**Interfaces:**
- Consumes: a bound `SessionWorkspaceBinding.RootPath` produced by Task 3.
- Produces: `workspace_required`, `workspace_path_escape`, and staged per-job request files under `.xelora/jobs/<job-id>/`.

- [ ] **Step 1: Replace the fallback test with failing enforcement and staging tests**

```go
func TestRunSkillScriptJobRequiresBoundWorkspace(t *testing.T) {
    gateway := NewGateway()
    _, err := gateway.RunSkillScriptJob(context.Background(), SkillJobRequest{SkillName: "writer", ScriptPath: "scripts/write.py"}, fakeExecutor)
    require.ErrorContains(t, err, "workspace_required")
}

func TestStagePreparedInputsMovesRequestIntoWorkspace(t *testing.T) {
    workspaceRoot := t.TempDir()
    skillRoot := t.TempDir()
    request := filepath.Join(skillRoot, "request.json")
    require.NoError(t, os.WriteFile(request, []byte(`{"action":"write"}`), 0o600))
    prepared := &skills.PreparedScriptExecution{BasePath: skillRoot, Args: []string{"request.json"}, MaterializedInputPaths: []string{request}}
    cleanup, err := stagePreparedInputs(prepared, workspaceRoot, "job-1")
    require.NoError(t, err)
    defer cleanup()
    require.Equal(t, filepath.Join(".xelora", "jobs", "job-1", "request.json"), prepared.Args[0])
}
```

- [ ] **Step 2: Run tests and verify failure**

Run: `go test ./internal/executor -run 'RequiresBoundWorkspace|StagePreparedInputs' -v`

Expected: FAIL because unbound jobs still fall back and request staging does not exist.

- [ ] **Step 3: Implement strict binding and staging**

```go
if !outputCtx.WriteAllowed || outputCtx.EffectiveRootDir == "" {
    code := outputCtx.FailureCode
    if code == "" { code = ConversationOutputFailureWorkspaceRequired }
    return nil, fmt.Errorf("%s: bind a workspace before running file tools", code)
}

func stagePreparedInputs(prepared *skills.PreparedScriptExecution, root, jobID string) (func(), error) {
    stageDir := filepath.Join(root, ".xelora", "jobs", jobID)
    if err := os.MkdirAll(stageDir, 0o700); err != nil { return nil, err }
    replacements := map[string]string{}
    for _, source := range prepared.MaterializedInputPaths {
        target := filepath.Join(stageDir, filepath.Base(source))
        if err := copyFile(source, target, 0o600); err != nil { return nil, err }
        replacements[source] = target
        replacements[filepath.Base(source)] = target
    }
    for i, arg := range prepared.Args {
        if target, ok := replacements[arg]; ok {
            rel, _ := filepath.Rel(root, target)
            prepared.Args[i] = rel
        }
    }
    staged := make([]string, 0, len(prepared.MaterializedInputPaths))
    for _, source := range prepared.MaterializedInputPaths {
        staged = append(staged, replacements[source])
    }
    prepared.MaterializedInputPaths = staged
    return func() { _ = os.RemoveAll(stageDir) }, nil
}
```

Create the job ID before preparation, stage inputs after preparation, set `prepared.WorkDir` to the canonical workspace root, and defer both cleanup functions. Make snapshot traversal ignore `.xelora`. Strengthen `IsWithinWorkspaceRoot` with `filepath.Rel` and `EvalSymlinks` for existing targets instead of string-prefix matching.

- [ ] **Step 4: Run executor and sandbox tests**

Run: `go test ./internal/executor ./internal/agent/skills ./internal/sandbox`

Expected: PASS; unbound jobs fail, bound jobs write to the workspace, staged inputs are cleaned, and `.xelora` files are not artifacts.

- [ ] **Step 5: Commit**

```powershell
git add internal/executor/gateway.go internal/executor/gateway_test.go internal/executor/types.go internal/agent/skills/manager.go
git commit -m "fix: enforce workspace-backed file execution"
```

### Task 5: Workspace-Aware File And Office Skills

**Files:**
- Modify: `skills/preloaded/workspace-file-writer/scripts/workspace_file_writer.py`
- Modify: `skills/preloaded/workspace-file-writer/scripts/workspace_file_writer_test.py`
- Modify: `skills/preloaded/officecli-document-editing/scripts/officecli_bridge.py`
- Modify: `skills/preloaded/officecli-document-editing/scripts/officecli_bridge_test.py`
- Modify: `skills/preloaded/workspace-file-writer/SKILL.md`
- Modify: `skills/preloaded/officecli-document-editing/SKILL.md`

**Interfaces:**
- Consumes: working directory and staged relative request path from Task 4.
- Produces: real workspace-relative outputs and post-edit Office validation.

- [ ] **Step 1: Add failing workspace and validation tests**

```python
def test_request_can_live_under_hidden_job_directory(tmp_path, monkeypatch):
    request = tmp_path / ".xelora" / "jobs" / "job-1" / "request.json"
    request.parent.mkdir(parents=True)
    request.write_text(json.dumps({"action": "write", "file": "report.md", "content": "ok"}), encoding="utf-8")
    monkeypatch.chdir(tmp_path)
    assert writer.main(["workspace_file_writer.py", ".xelora/jobs/job-1/request.json"]) == 0
    assert (tmp_path / "report.md").read_text(encoding="utf-8") == "ok"

def test_mutating_office_action_runs_validate(monkeypatch, tmp_path):
    calls = []
    monkeypatch.setattr(bridge.subprocess, "run", lambda command, **kwargs: calls.append(command) or Completed(0))
    bridge.execute({"action": "set", "file": "brief.docx", "path": "/body/p[1]", "props": {"text": "Updated"}}, tmp_path)
    assert calls[-1][:3] == ["officecli", "validate", "brief.docx"]

def test_failed_validation_preserves_original(monkeypatch, tmp_path):
    original = tmp_path / "brief.docx"
    original.write_bytes(b"original")
    monkeypatch.setattr(bridge, "run_officecli", fake_edit_then_failed_validation)
    assert bridge.execute(mutating_request("brief.docx"), tmp_path) != 0
    assert original.read_bytes() == b"original"
```

- [ ] **Step 2: Run tests and verify failure**

Run: `python -m unittest discover -s skills/preloaded/workspace-file-writer/scripts -p '*_test.py'`

Run: `python -m unittest discover -s skills/preloaded/officecli-document-editing/scripts -p '*_test.py'`

Expected: workspace writer test passes or exposes request-path assumptions; Office test FAILS because mutating commands do not validate.

- [ ] **Step 3: Separate request-path resolution from document-path resolution**

```python
def resolve_workspace_path(base_dir: Path, candidate: str) -> Path:
    path = (base_dir / candidate).resolve()
    path.relative_to(base_dir.resolve())
    return path

def is_mutating_action(action: str) -> bool:
    return action in {"create", "save", "remove", "set", "add", "batch"}

def execute_mutation(payload: dict, base_dir: Path) -> int:
    final_path = resolve_workspace_path(base_dir, require_field(payload, "file"))
    temp_path = final_path.with_name(f".{final_path.stem}.xelora-{uuid.uuid4().hex}{final_path.suffix}")
    if final_path.exists():
        shutil.copy2(final_path, temp_path)
    staged = dict(payload)
    staged["file"] = str(temp_path.relative_to(base_dir))
    command, batch_path = build_officecli_command(staged, base_dir)
    try:
        completed = run_officecli(command)
        if completed.returncode != 0:
            return completed.returncode
        validated = run_officecli(["officecli", "validate", staged["file"], "--json"])
        if validated.returncode != 0:
            print("office_validation_failed", file=sys.stderr)
            return validated.returncode
        os.replace(temp_path, final_path)
        return 0
    finally:
        if batch_path:
            Path(batch_path).unlink(missing_ok=True)
        temp_path.unlink(missing_ok=True)
```

Use `execute_mutation` for `create`, `save`, `remove`, `set`, `add`, and `batch`; read-only actions continue directly. Both scripts reject absolute document paths and traversal. Update skill docs to state that paths are relative to the bound conversation workspace and that an unbound call fails before script execution.

- [ ] **Step 4: Run Python tests and Docker smoke scripts against a temporary bound directory**

Run: `python -m unittest discover -s skills/preloaded/workspace-file-writer/scripts -p '*_test.py'`

Run: `python -m unittest discover -s skills/preloaded/officecli-document-editing/scripts -p '*_test.py'`

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add skills/preloaded/workspace-file-writer skills/preloaded/officecli-document-editing
git commit -m "feat: make file skills workspace aware"
```

### Task 6: New Conversation Workspace Selector

**Files:**
- Create: `frontend/src/api/workspace/index.ts`
- Create: `frontend/src/components/WorkspaceSelector.vue`
- Modify: `frontend/src/views/creatChat/creatChat.vue`
- Modify: `frontend/src/api/chat/index.ts`
- Modify: `frontend/src/i18n/locales/en-US.ts`
- Modify: `frontend/src/i18n/locales/zh-CN.ts`

**Interfaces:**
- Consumes: Task 2 workspace API and Task 3 `workspace_binding: {workspace_id}` request shape.
- Produces: required workspace selection in the first-message session creation flow and persisted recent selection in local storage.

- [ ] **Step 1: Add API types and a component testable contract**

```ts
export interface WorkspaceEntry {
  id: string;
  name: string;
  relative_path: string;
  status: 'available' | 'missing' | 'access_denied' | 'archived';
}

export const listWorkspaces = () => get<WorkspaceEntry[]>('/api/v1/workspaces');
export const createWorkspace = (name: string) => post<WorkspaceEntry>('/api/v1/workspaces', { name });
```

Run: `npm run type-check`

Expected: FAIL until component state and chat payload types are updated; record unrelated pre-existing TypeScript failures separately.

- [ ] **Step 2: Implement `WorkspaceSelector.vue`**

```vue
<t-select v-model="selectedId" :placeholder="t('workspace.select')" @change="emitSelection">
  <t-option v-for="item in available" :key="item.id" :value="item.id" :label="item.name" />
</t-select>
<t-button variant="text" theme="primary" @click="showCreate = true">
  <template #icon><folder-add-icon /></template>{{ t('workspace.newFolder') }}
</t-button>
```

The create dialog accepts one folder name, calls `createWorkspace`, selects the result, and shows backend errors through `MessagePlugin`. Emit `update:modelValue` only for `available` entries. Store the last successful ID as `xelora:last-workspace-id` and ignore it if it is no longer listed.

- [ ] **Step 3: Replace automatic `tenant:<id>` binding**

```ts
const selectedWorkspaceId = ref<string>('');

if (selectedWorkspaceId.value) {
  sessionData.workspace_binding = { workspace_id: selectedWorkspaceId.value };
}
```

Render `WorkspaceSelector` beside the agent/model controls. Do not send `workspace_name` or `root_path`. Keep plain chat creation possible without selection; file tools enforce the requirement later.

- [ ] **Step 4: Build and inspect responsive UI**

Run: `npm run build-only`

Expected: PASS. Use the local app at desktop and mobile widths to verify selector text does not overlap model or agent controls and the create-folder dialog remains usable.

- [ ] **Step 5: Commit**

```powershell
git add frontend/src/api/workspace/index.ts frontend/src/components/WorkspaceSelector.vue frontend/src/views/creatChat/creatChat.vue frontend/src/api/chat/index.ts frontend/src/i18n/locales/en-US.ts frontend/src/i18n/locales/zh-CN.ts
git commit -m "feat: select workspace for new conversations"
```

### Task 7: Docker Configuration And End-To-End Smoke

**Files:**
- Modify: `.env.example`
- Modify: `docker-compose.yml`
- Create: `scripts/host-workspace-mount-smoke.ps1`
- Modify: `docs/api/session.md`
- Modify: `docs/customizations/TASKS.md`

**Interfaces:**
- Consumes: all prior tasks.
- Produces: a reproducible Windows Docker Desktop deployment and end-to-end proof.

- [ ] **Step 1: Add Compose configuration and validate interpolation**

```yaml
services:
  app:
    volumes:
      - ${XELORA_WORKSPACE_HOST_ROOT:-./data/workspaces}:${XELORA_WORKSPACE_CONTAINER_ROOT:-/workspaces}:rw
    environment:
      - XELORA_WORKSPACE_CONTAINER_ROOT=${XELORA_WORKSPACE_CONTAINER_ROOT:-/workspaces}
```

Document:

```dotenv
XELORA_WORKSPACE_HOST_ROOT=F:\XeloraWorkspaces
XELORA_WORKSPACE_CONTAINER_ROOT=/workspaces
```

Run: `docker compose config --quiet`

Expected: exit code 0 and the app service contains exactly one `/workspaces` mount.

- [ ] **Step 2: Add a live smoke script**

```powershell
param([string]$HostRoot = (Join-Path $PSScriptRoot '..\data\workspaces-e2e'))
$ErrorActionPreference = 'Stop'
$resolvedRoot = (New-Item -ItemType Directory -Force $HostRoot).FullName
$env:XELORA_WORKSPACE_HOST_ROOT = $resolvedRoot
docker compose up -d --build app frontend
docker compose exec -T app sh -lc 'test -w /workspaces && printf workspace-mount-ok > /workspaces/.mount-smoke'
$hostProbe = Join-Path $resolvedRoot '.mount-smoke'
if (-not (Test-Path -LiteralPath $hostProbe)) { throw 'workspace probe was not visible on the Windows host' }
if ((Get-Content -Raw -LiteralPath $hostProbe) -ne 'workspace-mount-ok') { throw 'workspace probe content mismatch' }
Remove-Item -LiteralPath $hostProbe
Write-Host "Host workspace mount passed: $resolvedRoot"
```

This script proves the bidirectional Docker Desktop mount without depending on browser authentication. The authenticated conversation and Office workflow remains the explicit Chrome acceptance step below.

- [ ] **Step 3: Run backend, frontend, and Python suites**

Run: `docker run --rm -v "${PWD}:/src" -w /src golang:1.26 go test ./internal/workspace ./internal/handler ./internal/handler/session ./internal/application/service ./internal/application/repository ./internal/executor ./internal/agent/skills ./internal/sandbox`

Run: `cd frontend; npm run build-only`

Run: `python -m unittest discover -s skills/preloaded/workspace-file-writer/scripts -p '*_test.py'`

Run: `python -m unittest discover -s skills/preloaded/officecli-document-editing/scripts -p '*_test.py'`

Expected: all targeted suites and frontend build PASS.

- [ ] **Step 4: Perform browser acceptance using the existing Chrome login session**

Create a workspace folder, create a new conversation bound to it, create and edit a Markdown file, then create and edit one `.docx`, `.xlsx`, and `.pptx`. Reopen the conversation and modify an existing Office file. Verify every file exists under the Windows host root and no output appears under a skill-private directory. Attempt `../escape.md` and an unbound file-producing request and verify `workspace_path_escape` and `workspace_required` respectively.

- [ ] **Step 5: Update docs and task board with evidence**

Document the workspace API request/response shape, Docker variables, legacy binding behavior, smoke command, and exact test results. Mark the related task complete only after browser and host-filesystem evidence both pass.

- [ ] **Step 6: Commit**

```powershell
git add .env.example docker-compose.yml scripts/host-workspace-mount-smoke.ps1 docs/api/session.md docs/customizations/TASKS.md
git commit -m "docs: add host workspace deployment and smoke test"
```

## Completion Audit

- A selected workspace is represented by a server-issued ID and canonical path never comes from the client.
- New and reopened conversations preserve the same binding.
- Unbound file work fails instead of writing under `skills/preloaded`.
- Request JSON is staged under `.xelora/jobs`, cleaned after execution, and excluded from artifacts.
- Markdown, JSON, CSV, DOCX, XLSX, PPTX, previews, and browser captures share the same workspace/artifact path.
- Same-conversation and reopened-conversation edits modify the existing host file.
- Traversal and symlink escapes fail before execution.
- Docker Desktop host files provide authoritative persistence evidence.
