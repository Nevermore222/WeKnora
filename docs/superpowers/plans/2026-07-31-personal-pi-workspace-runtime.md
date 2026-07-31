# Personal Pi Workspace Runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** On Personal, route workspace Skill / file / shell tool execution through a Pi coding-agent sidecar while keeping the Go agent loop and KB tools unchanged.

**Architecture:** Go remains the only decision loop. A Workspace Tool Facade resolves `sessions.workspace_binding.root_path`, enforces path rules and mutation confirm, then calls a versioned stdio RPC to a Node Pi sidecar. Non-Personal editions never start the sidecar and keep the current Go executor path.

**Tech Stack:** Go (existing agent tools + executor), Node/TypeScript Pi coding-agent SDK sidecar, Wails Personal lifecycle, existing mutation confirm SSE/UI.

**Spec:** `docs/superpowers/specs/2026-07-31-personal-pi-workspace-runtime-design.md`

**Progress (2026-07-31):** Tasks 1–8 implemented on `002-session-workspace-binding` (HEAD includes `c05c0969` runtime + `6096d4de` sticky Personal-mode fix). Skill script routing via Pi remains deferred; confirm mutations stay on Go apply path.

---

## File map

| File | Responsibility |
|---|---|
| `internal/piruntime/protocol.go` | Versioned RPC request/response types + error codes |
| `internal/piruntime/client.go` | Stdio client: start/stop, handshake, Call |
| `internal/piruntime/client_test.go` | Fake sidecar process tests |
| `internal/piruntime/facade.go` | Personal gate, cwd inject, map to `types.ToolResult` |
| `internal/piruntime/facade_test.go` | Non-Personal short-circuit, unbound, path escape |
| `internal/piruntime/lifecycle.go` | Process manager used by desktop/container |
| `sidecars/pi-workspace/package.json` | Sidecar package + pinned Pi dependency |
| `sidecars/pi-workspace/src/main.ts` | Stdio RPC server loop |
| `sidecars/pi-workspace/src/tools.ts` | cwd-scoped file/shell/skill ops via Pi factories or fs |
| `sidecars/pi-workspace/src/protocol.ts` | Mirror Go protocol types |
| `cmd/desktop/app.go` | Start/stop sidecar on Personal startup/shutdown |
| `internal/agent/tools/definitions.go` | Add `workspace_read/write/edit/list/shell` constants |
| `internal/agent/tools/workspace_fs.go` | New file/shell tools → Facade |
| `internal/agent/tools/skill_execute.go` | Personal path: Facade instead of Go executor gateway |
| `internal/application/service/agent_service.go` | Register new tools; wire Facade + resolver |
| `scripts/build-personal.ps1` | Bundle sidecar into `dist/personal` |
| `docs/superpowers/specs/2026-07-31-personal-pi-workspace-runtime-design.md` | Source of truth (already committed) |

---

### Task 1: RPC protocol types (Go)

**Files:**
- Create: `internal/piruntime/protocol.go`
- Create: `internal/piruntime/protocol_test.go`

- [ ] **Step 1: Write failing marshal tests**

```go
package piruntime_test

import (
	"encoding/json"
	"testing"

	"github.com/Tencent/Xelora/internal/piruntime"
	"github.com/stretchr/testify/require"
)

func TestRequestJSON_RoundTrip(t *testing.T) {
	req := piruntime.Request{
		ProtocolVersion: piruntime.ProtocolVersion,
		CorrelationID:   "c1",
		SessionID:       "s1",
		ToolName:        "workspace_read",
		Args:            json.RawMessage(`{"path":"a.txt"}`),
		Cwd:             `D:\proj`,
		TimeoutMs:       5000,
	}
	b, err := json.Marshal(req)
	require.NoError(t, err)
	var got piruntime.Request
	require.NoError(t, json.Unmarshal(b, &got))
	require.Equal(t, piruntime.ProtocolVersion, got.ProtocolVersion)
	require.Equal(t, "workspace_read", got.ToolName)
	require.Equal(t, `D:\proj`, got.Cwd)
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./internal/piruntime/ -run TestRequestJSON -count=1
```

Expected: package not found / undefined types

- [ ] **Step 3: Implement protocol**

```go
package piruntime

import "encoding/json"

const ProtocolVersion = 1

const (
	ErrRuntimeUnavailable   = "runtime_unavailable"
	ErrPathOutOfWorkspace   = "path_out_of_workspace"
	ErrProtocolMismatch     = "protocol_mismatch"
	ErrTimeout              = "timeout"
	ErrWorkspaceRequired    = "workspace_required"
	ErrBindingInvalid       = "binding_invalid"
	ErrMutationRejected     = "mutation_rejected"
)

type Request struct {
	ProtocolVersion int             `json:"protocol_version"`
	CorrelationID   string          `json:"correlation_id"`
	SessionID       string          `json:"session_id"`
	ToolName        string          `json:"tool_name"`
	Args            json.RawMessage `json:"args"`
	Cwd             string          `json:"cwd"`
	PermissionMode  string          `json:"permission_mode,omitempty"`
	TimeoutMs       int             `json:"timeout_ms"`
}

type Response struct {
	OK           bool              `json:"ok"`
	ErrorCode    string            `json:"error_code,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	Content      string            `json:"content,omitempty"`
	Artifacts    []string          `json:"artifacts,omitempty"`
	DurationMs   int64             `json:"duration_ms,omitempty"`
	Data         map[string]any    `json:"data,omitempty"`
}
```

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/piruntime/ -run TestRequestJSON -count=1
```

- [ ] **Step 5: Commit**

```bash
git add internal/piruntime/protocol.go internal/piruntime/protocol_test.go
# message: feat: Pi workspace runtime RPC protocol types
```

---

### Task 2: Stdio client + fake sidecar tests

**Files:**
- Create: `internal/piruntime/client.go`
- Create: `internal/piruntime/client_test.go`

- [ ] **Step 1: Write failing client test using a fake Node/Python-less echo script**

Use a Go test helper that spawns `go run` of a tiny `internal/piruntime/testdata/fake_sidecar.go` which reads one JSON line from stdin and writes one JSON line to stdout.

```go
func TestClient_CallEcho(t *testing.T) {
	cli, err := piruntime.Start(context.Background(), piruntime.StartOptions{
		Command: os.Args[0],
		Args:    []string{"-test.run=TestFakeSidecarHelper", "--", "sidecar"},
		Env:     []string{"PI_FAKE_SIDECAR=1"},
	})
	// Prefer: exec helper binary built in TestMain, or `go run ./internal/piruntime/testdata/fake_sidecar.go`
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })
	require.NoError(t, cli.Handshake(context.Background()))
	resp, err := cli.Call(context.Background(), piruntime.Request{
		ProtocolVersion: piruntime.ProtocolVersion,
		CorrelationID:   "1",
		ToolName:        "workspace_read",
		Args:            json.RawMessage(`{"path":"a.txt"}`),
		Cwd:             t.TempDir(),
		TimeoutMs:       3000,
	})
	require.NoError(t, err)
	require.True(t, resp.OK)
	require.Contains(t, resp.Content, "a.txt")
}
```

Implement `testdata/fake_sidecar.go` as a standalone `package main` that:
1. On stdin line `{"type":"handshake"}` → stdout `{"type":"handshake_ok","protocol_version":1}`
2. On stdin request JSON → stdout Response `{ok:true, content:"echo:"+tool_name+path}`

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./internal/piruntime/ -run TestClient_CallEcho -count=1
```

- [ ] **Step 3: Implement client**

```go
// Client owns cmd, stdin pipe, stdout scanner, mu.
// Start(ctx, StartOptions{Command, Args, Env}) (*Client, error)
// Handshake(ctx) error — protocol_version must match
// Call(ctx, Request) (Response, error) — one JSON line request / one JSON line response
// Close() error — kill process
```

Framing: **NDJSON** (one JSON object per line), UTF-8. Mutex so Call is serialized.

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/piruntime/ -count=1
```

- [ ] **Step 5: Commit**

```bash
git add internal/piruntime/
# message: feat: Pi workspace stdio RPC client
```

---

### Task 3: Facade (Personal gate + cwd + ToolResult map)

**Files:**
- Create: `internal/piruntime/facade.go`
- Create: `internal/piruntime/facade_test.go`

- [ ] **Step 1: Failing tests**

```go
func TestFacade_NonPersonalUsesFallback(t *testing.T) {
	workspace.SetPersonalLocalFoldersEnabled(false)
	t.Cleanup(func() { workspace.SetPersonalLocalFoldersEnabled(false) })
	called := false
	f := piruntime.NewFacade(nil, func(ctx context.Context, tool string, args json.RawMessage) (*types.ToolResult, error) {
		called = true
		return &types.ToolResult{Success: true, Output: "go"}, nil
	})
	res, err := f.Execute(context.Background(), piruntime.ExecuteInput{
		SessionID: "s",
		ToolName:  "execute_skill_script",
		Args:      json.RawMessage(`{}`),
		Binding:   &types.SessionWorkspaceBinding{Status: types.SessionWorkspaceBindingStatusBound, RootPath: t.TempDir()},
	})
	require.NoError(t, err)
	require.True(t, called)
	require.Equal(t, "go", res.Output)
}

func TestFacade_UnboundFails(t *testing.T) {
	workspace.SetPersonalLocalFoldersEnabled(true)
	t.Cleanup(func() { workspace.SetPersonalLocalFoldersEnabled(false) })
	f := piruntime.NewFacade(&fakeClient{}, nil)
	_, err := f.Execute(context.Background(), piruntime.ExecuteInput{
		ToolName: "workspace_read",
		Args:     json.RawMessage(`{"path":"a"}`),
		Binding:  nil,
	})
	require.ErrorContains(t, err, "workspace_required")
}

func TestFacade_InjectsCwdFromBinding(t *testing.T) {
	workspace.SetPersonalLocalFoldersEnabled(true)
	t.Cleanup(func() { workspace.SetPersonalLocalFoldersEnabled(false) })
	root := t.TempDir()
	fc := &fakeClient{}
	f := piruntime.NewFacade(fc, nil)
	_, err := f.Execute(context.Background(), piruntime.ExecuteInput{
		SessionID: "s",
		ToolName:  "workspace_read",
		Args:      json.RawMessage(`{"path":"a.txt"}`),
		Binding: &types.SessionWorkspaceBinding{
			Status:   types.SessionWorkspaceBindingStatusBound,
			RootPath: root,
		},
	})
	require.NoError(t, err)
	require.Equal(t, root, fc.last.Cwd)
	require.Equal(t, piruntime.ProtocolVersion, fc.last.ProtocolVersion)
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./internal/piruntime/ -run TestFacade -count=1
```

- [ ] **Step 3: Implement Facade**

```go
type ExecuteInput struct {
	SessionID string
	ToolName  string
	Args      json.RawMessage
	Binding   *types.SessionWorkspaceBinding
	TimeoutMs int
}

type Facade struct {
	client   Caller // interface { Handshake; Call; Close }
	fallback func(ctx context.Context, tool string, args json.RawMessage) (*types.ToolResult, error)
}

func (f *Facade) Execute(ctx context.Context, in ExecuteInput) (*types.ToolResult, error) {
	if !workspace.PersonalLocalFoldersEnabled() {
		if f.fallback == nil {
			return &types.ToolResult{Success: false, Error: ErrRuntimeUnavailable}, nil
		}
		return f.fallback(ctx, in.ToolName, in.Args)
	}
	if in.Binding == nil || in.Binding.Status != types.SessionWorkspaceBindingStatusBound || strings.TrimSpace(in.Binding.RootPath) == "" {
		return &types.ToolResult{Success: false, Error: ErrWorkspaceRequired + ": bind a workspace before running file tools"}, nil
	}
	if st, err := os.Stat(in.Binding.RootPath); err != nil || !st.IsDir() {
		return &types.ToolResult{Success: false, Error: ErrBindingInvalid + ": root_path missing"}, nil
	}
	resp, err := f.client.Call(ctx, Request{ /* fill */ Cwd: in.Binding.RootPath })
	// map resp → ToolResult
}
```

Never pass model-supplied root as cwd.

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/piruntime/ -count=1
```

- [ ] **Step 5: Commit**

```bash
git add internal/piruntime/
# message: feat: Personal Pi workspace tool facade
```

---

### Task 4: Minimal Pi sidecar (Node) + handshake

**Files:**
- Create: `sidecars/pi-workspace/package.json`
- Create: `sidecars/pi-workspace/tsconfig.json`
- Create: `sidecars/pi-workspace/src/protocol.ts`
- Create: `sidecars/pi-workspace/src/main.ts`
- Create: `sidecars/pi-workspace/src/tools.ts`
- Create: `sidecars/pi-workspace/README.md`

- [ ] **Step 1: Scaffold package**

```json
{
  "name": "xelora-pi-workspace-sidecar",
  "private": true,
  "type": "module",
  "engines": { "node": ">=22" },
  "scripts": {
    "build": "tsc -p tsconfig.json",
    "start": "node dist/main.js"
  },
  "dependencies": {
    "@mariozechner/pi-coding-agent": "latest"
  },
  "devDependencies": {
    "typescript": "^5.8.0",
    "@types/node": "^22.0.0"
  }
}
```

Pin the real Pi package version after `npm view` during implementation (prefer `@earendil-works/pi-coding-agent` or current published name from pi.dev docs — **verify at implement time and lock exact version**).

- [ ] **Step 2: Implement NDJSON loop in `main.ts`**

- Read lines from stdin
- `handshake` → `{type:"handshake_ok", protocol_version:1}`
- Otherwise parse `Request`, dispatch `tools.handle(req)`, write `Response` line
- On unknown tool → `{ok:false, error_code:"unknown_tool"}`

- [ ] **Step 3: Implement `tools.ts` (cwd confined)**

First vertical slice without full Pi SDK wiring if package API discovery takes time — use Node `fs/promises` + `child_process` with path confinement:

```ts
function resolveInCwd(cwd: string, rel: string): string {
  const abs = path.resolve(cwd, rel);
  const root = path.resolve(cwd);
  if (abs !== root && !abs.startsWith(root + path.sep)) {
    throw Object.assign(new Error("path_out_of_workspace"), { code: "path_out_of_workspace" });
  }
  return abs;
}
```

Tools:
- `workspace_read` `{path}`
- `workspace_write` `{path, content}`
- `workspace_edit` `{path, old_text, new_text}` (single replace)
- `workspace_list` `{path?}`
- `workspace_shell` `{command}` — `spawn` with `cwd`, `shell:true` on win32, timeout
- `execute_skill_script` `{skill_name, script, args?}` — run script file under cwd/skills layout; phase-1 can exec `node`/`bash` on resolved script path
- `workspace_mutate` apply-only ops `{op, from, to?}` after Go already confirmed (optional; Go may keep apply locally)

- [ ] **Step 4: Manual stdio smoke**

```bash
cd sidecars/pi-workspace && npm install && npm run build
# pipe handshake + read
```

- [ ] **Step 5: Commit**

```bash
git add sidecars/pi-workspace/
# message: feat: Pi workspace sidecar NDJSON runtime
```

**Note:** Integrate real Pi tool factories in a follow-up commit inside this task if the SDK import surface is clear; confinement tests must pass either way.

---

### Task 5: Desktop lifecycle (start/stop sidecar)

**Files:**
- Create: `internal/piruntime/lifecycle.go`
- Modify: `cmd/desktop/app.go`
- Modify: `scripts/build-personal.ps1` (copy `sidecars/pi-workspace/dist` + `node_modules` or bundled node)

- [ ] **Step 1: Lifecycle API**

```go
// Manager holds *Client
// StartPersonal(ctx, sidecarExeOrNodeScript) error — no-op if !PersonalLocalFoldersEnabled
// Default() *Facade
// Stop(ctx) error
```

Wire global/accessor used by `agent_service` (same pattern as other desktop singletons), or inject via container when `Edition=="personal"`.

- [ ] **Step 2: `app.startup` / `shutdown`**

```go
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if err := piruntime.StartFromPersonalBundle(ctx); err != nil {
		// log only — tools will surface runtime_unavailable
	}
}

func (a *App) shutdown(ctx context.Context) {
	_ = piruntime.Stop(ctx)
	a.shutdownCh <- struct{}{}
}
```

- [ ] **Step 3: build-personal.ps1**

After frontend/wails build, `npm ci && npm run build` in `sidecars/pi-workspace`, copy to `dist/personal/pi-workspace/`. Document Node requirement: embed portable Node **or** require system Node ≥22 for v1 (choose embed if build script already vendors runtimes; otherwise document system Node and fail handshake clearly).

- [ ] **Step 4: Commit**

```bash
# message: feat: start Pi workspace sidecar with Personal desktop
```

---

### Task 6: Agent tools — file/shell + route skill execute

**Files:**
- Modify: `internal/agent/tools/definitions.go`
- Create: `internal/agent/tools/workspace_fs.go`
- Create: `internal/agent/tools/workspace_fs_test.go`
- Modify: `internal/agent/tools/skill_execute.go`
- Modify: `internal/application/service/agent_service.go`

- [ ] **Step 1: Add tool name constants**

```go
ToolWorkspaceRead  = "workspace_read"
ToolWorkspaceWrite = "workspace_write"
ToolWorkspaceEdit  = "workspace_edit"
ToolWorkspaceList  = "workspace_list"
ToolWorkspaceShell = "workspace_shell"
```

Add to `AvailableToolDefinitions` / Personal default allowlist only when `PersonalLocalFoldersEnabled()` (register in `registerTools` gated the same way as `workspace_mutate`).

- [ ] **Step 2: Implement tools calling Facade**

Each tool: parse args → `facade.Execute` with binding from `WithSessionWorkspaceResolver` (same pattern as `ExecuteSkillScriptTool`).

- [ ] **Step 3: `ExecuteSkillScriptTool`**

```go
if workspace.PersonalLocalFoldersEnabled() && facade != nil {
  return facade.Execute(ctx, piruntime.ExecuteInput{
    ToolName: ToolExecuteSkillScript,
    Args: args,
    Binding: resolver(meta.SessionID),
  })
}
// existing gateway path
```

- [ ] **Step 4: Unit tests** — mock Facade/Caller; assert non-Personal skill path still hits gateway (existing tests remain green).

```bash
go test ./internal/agent/tools/ ./internal/piruntime/ ./internal/executor/ -count=1
```

- [ ] **Step 5: Commit**

```bash
# message: feat: route Personal workspace file/shell/skill tools via Pi facade
```

---

### Task 7: Destructive ops stay on confirm path

**Files:**
- Modify: `internal/agent/tools/workspace_mutate.go` (optional apply via Facade)
- Modify: `internal/workspace/mutation.go` only if apply delegated to Pi

- [ ] **Step 1: Keep propose → SSE → UI confirm → apply as today**

- [ ] **Step 2: After user apply, either**
  - (A) keep Go `MutationStore.Apply` (already path-confined) — **preferred for v1**, or
  - (B) call sidecar `workspace_mutate` with `permission_mode=elevated`

Choose **(A)** unless Apply duplication becomes painful. Document choice in commit message.

- [ ] **Step 3: Test reject/approve still works**

```bash
go test ./internal/agent/tools/ -run WorkspaceMutate -count=1
go test ./internal/workspace/ -run Mutation -count=1
```

- [ ] **Step 4: Commit**

```bash
# message: fix: keep workspace mutation confirm on Go path with Pi file tools
```

---

### Task 8: Verification

**Files:** none beyond fixes

- [ ] **Step 1: Unit suite**

```bash
go test ./internal/piruntime/ ./internal/agent/tools/ ./internal/workspace/ ./internal/executor/ ./internal/application/service/ -count=1
```

Expected: PASS (KB/session tests unchanged in behavior)

- [ ] **Step 2: Rebuild Personal**

```powershell
Get-Process -Name "Xelora Personal" -ErrorAction SilentlyContinue | Stop-Process -Force
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-personal.ps1
```

- [ ] **Step 3: Manual checklist**

1. Open Folder → chat → `workspace_list` / ask AI to list files  
2. Write a file via agent → appears on disk under root  
3. Run a skill script that writes under cwd  
4. Shell `echo` / `dir` under cwd  
5. Delete/rename → confirm UI → disk updates only after confirm  
6. Reject mutation → unchanged  
7. Ask a KB question in same session → Progressive RAG still works  
8. Quit app → no leftover `node` sidecar in Task Manager  

- [ ] **Step 4: Commit fixes if any**

```bash
# message: fix: Personal Pi workspace runtime verification follow-ups
```

---

## Spec coverage

| Spec item | Task |
|---|---|
| Go main loop + Pi tool backend | 3, 6 |
| Skill + file + shell via Pi | 4, 6 |
| Personal only | 3, 5 |
| KB unchanged | 6 (no KB tool edits), 8 regression |
| stdio RPC + version pin | 1, 2, 4 |
| cwd from binding only | 3 |
| Mutation confirm UX | 7 |
| Lifecycle / packaging | 5 |
| Error codes / no silent fallback | 3, 4 |
| Success criteria / smoke | 8 |

## Placeholder / consistency notes

- Exact npm package name for Pi must be verified at Task 4 implement time and locked in `package.json`.
- v1 sidecar may use confined Node fs/child_process first; swapping internals to Pi SDK factories must preserve the same RPC tool names.
- `workspace_mutate` apply remains Go-side in v1 (Task 7A).
