# Internal Client Server Workspace Feature Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a server-controlled client workspace feature switch that hides the existing server-workspace UI in the first internal client release while preserving server binding, Controlled Docker Skill execution, artifacts, and existing account permissions.

**Architecture:** Keep the existing Go Agent loop, Progressive RAG, session workspace binding, Executor Gateway, Controlled Docker provider, and Skill Manager unchanged. Add a dynamic authenticated client-feature response backed by the existing system-settings registry; the enterprise desktop client loads that response into a small Pinia store and gates only the existing workspace UI and new-session payload. Package the internal client with an immutable build-time default server origin while preserving the current development fallback for ordinary Personal builds.

**Tech Stack:** Go/Gin, GORM-backed `system_settings`, existing RBAC middleware, Vue 3, Pinia, TypeScript, Wails desktop bootstrap, Node test runner, Go unit tests, existing Controlled Docker smoke path.

---

## Repository Constraints

- Preserve the dirty worktree and inspect overlapping changes before each edit.
- Do not touch, stage, or commit `n8n`, CL, COBOL, or unrelated workflow content.
- Do not change the existing Go Agent loop, Progressive RAG prompt construction,
  retrieval tools, `Executor Gateway`, or Skill execution provider behavior.
- Do not introduce local `root_path` binding into the internal enterprise
  client. The client sends only server `workspace_id` values.
- Do not add commit steps to this plan. The user requires selective Git
  delivery; implementation changes must be reviewed and explicitly staged
  later.

## File Map

Create:

- `internal/handler/client_features.go` - dynamic authenticated client-feature
  response.
- `internal/handler/client_features_test.go` - response and fail-closed tests.
- `internal/application/service/system_setting_workspace_test.go` - setting
  registry and default test.
- `frontend/src/stores/clientFeatures.ts` - runtime-scoped feature state.
- `frontend/src/stores/clientFeatures.test.ts` - feature store tests.
- `frontend/src/utils/clientWorkspaceFeature.ts` - pure visibility and payload
  helpers.
- `frontend/src/utils/clientWorkspaceFeature.test.ts` - helper tests.

Modify:

- `internal/application/service/system_setting.go` - register the default-off
  boolean setting.
- `internal/router/router.go` - register the authenticated client-feature
  route under `RegisterSystemRoutes`.
- `frontend/src/api/system/capabilities.ts` - fetch and type dynamic features.
- `frontend/src/App.vue` - refresh features after login and reset on logout.
- `frontend/src/stores/enterprise.ts` - refresh after enterprise profile switch.
- `frontend/src/views/creatChat/creatChat.vue` - gate selector and payload.
- `frontend/src/views/chat/index.vue` - gate display and file-browser controls.
- `frontend/src/i18n/locales/zh-CN.ts` - administrator setting translations.
- `frontend/src/i18n/locales/en-US.ts` - administrator setting translations.
- `cmd/desktop/personal_defaults.go` - packaged server-origin precedence.
- `cmd/desktop/desktop_env_test.go` - packaged-origin test.
- `scripts/build-personal.ps1` - internal server-origin build parameter.
- `frontend/src/utils/default-enterprise-bootstrap.ts` - preserve bootstrap
  behavior for the packaged origin.
- `frontend/src/utils/default-enterprise-bootstrap.test.ts` - bootstrap tests.

## Task 1: Register The Server Feature Setting

**Files:**

- Modify: `internal/application/service/system_setting.go`
- Create: `internal/application/service/system_setting_workspace_test.go`
- Modify: `frontend/src/i18n/locales/zh-CN.ts`
- Modify: `frontend/src/i18n/locales/en-US.ts`

- [x] **Step 1: Write the failing registry test**

Add a focused test that loads the existing setting registry and asserts the
exact key, boolean type, and false default:

```go
func TestClientWorkspaceBindingSettingDefaultsOff(t *testing.T) {
    item, ok := systemSettingDefinitions["client.workspace_binding_enabled"]
    if !ok {
        t.Fatal("missing client workspace binding setting")
    }
    if item.Type != "bool" {
        t.Fatalf("type=%q, want bool", item.Type)
    }
    if item.Default != false {
        t.Fatalf("default=%v, want false", item.Default)
    }
}
```

Use the registry symbol and default representation already present in
`internal/application/service/system_setting.go`; keep this test in package
`service` if the registry is unexported.

- [x] **Step 2: Run the test and verify it fails**

Run:

```powershell
go test ./internal/application/service -run TestClientWorkspaceBindingSettingDefaultsOff -count=1
```

Expected: FAIL because the setting is not registered.

- [x] **Step 3: Add the default-off setting**

Register the key beside the existing system settings:

```go
"client.workspace_binding_enabled": {
    Type:        "bool",
    EnvName:     "XELORA_CLIENT_WORKSPACE_BINDING_ENABLED",
    Default:     false,
    Category:    "client",
    Description: "Whether enterprise desktop clients may show server workspace binding controls.",
},
```

Use the existing database > environment > built-in default precedence and audit
path. Do not make this compile-time-only.

- [x] **Step 4: Add operator translations**

Add Chinese and English labels and descriptions for
`client.workspace_binding_enabled`. Explain that the value controls enterprise
client workspace UI and does not replace server authorization.

- [x] **Step 5: Run focused setting tests**

Run:

```powershell
go test ./internal/application/service -run 'Test.*(SystemSetting|ClientWorkspace)' -count=1
```

Expected: PASS, including the new false default.

## Task 2: Expose Dynamic Client Features After Authentication

**Files:**

- Create: `internal/handler/client_features.go`
- Create: `internal/handler/client_features_test.go`
- Modify: `internal/router/router.go`
- Modify: `frontend/src/api/system/capabilities.ts`

- [x] **Step 1: Write handler contract tests**

Use the existing `SystemHandler` construction and a small
`SystemSettingService` stub. Cover false, true, and malformed values:

```go
func TestGetClientFeaturesDefaultsWorkspaceBindingOff(t *testing.T) {
    h := &SystemHandler{systemSettingSvc: settingStub{value: "false"}}
    w := callClientFeatures(h)
    require.Equal(t, http.StatusOK, w.Code)
    require.JSONEq(t, `{"features":{"workspace_binding":false}}`, w.Body.String())
}

func TestGetClientFeaturesReturnsEnabledValue(t *testing.T) {
    h := &SystemHandler{systemSettingSvc: settingStub{value: "true"}}
    w := callClientFeatures(h)
    require.JSONEq(t, `{"features":{"workspace_binding":true}}`, w.Body.String())
}

func TestGetClientFeaturesFailsClosedForMalformedValue(t *testing.T) {
    h := &SystemHandler{systemSettingSvc: settingStub{value: "not-a-bool"}}
    w := callClientFeatures(h)
    require.JSONEq(t, `{"features":{"workspace_binding":false}}`, w.Body.String())
}
```

The helper must attach the same authenticated viewer context used by system
route tests; this endpoint is not public.

- [x] **Step 2: Run the tests and verify they fail**

Run:

```powershell
go test ./internal/handler -run TestGetClientFeatures -count=1
```

Expected: FAIL because the handler and route do not exist.

- [x] **Step 3: Implement the dynamic response**

Add `GetClientFeatures` as a method on the existing `SystemHandler` in
`internal/handler/client_features.go`:

```go
type ClientFeaturesResponse struct {
    Features map[string]bool `json:"features"`
}

func (h *SystemHandler) GetClientFeatures(c *gin.Context) {
    enabled := false
    if h.systemSettingSvc != nil {
        raw := strings.TrimSpace(h.systemSettingSvc.GetString(
            c.Request.Context(),
            "client.workspace_binding_enabled",
            "",
            "false",
        ))
        enabled = strings.EqualFold(raw, "true") || raw == "1"
    }
    c.JSON(http.StatusOK, ClientFeaturesResponse{
        Features: map[string]bool{"workspace_binding": enabled},
    })
}
```

Keep this separate from the public `/api/v1/system/capabilities` response,
whose static `features: []string` contract is used for API negotiation.

- [x] **Step 4: Register the authenticated route**

Inside `RegisterSystemRoutes` in `internal/router/router.go`, add:

```go
systemRoutes.GET("/client-features", g.Viewer(), handler.GetClientFeatures)
```

The path is `/api/v1/system/client-features`. Do not add a new admin write
endpoint; the existing system-settings admin routes provide authorization,
persistence, and audit logging for the registered key.

- [x] **Step 5: Add the TypeScript API contract**

Extend `frontend/src/api/system/capabilities.ts` with:

```ts
export interface ClientFeaturesResponse {
  features: {
    workspace_binding?: boolean
  }
}

export async function getClientFeatures(): Promise<ClientFeaturesResponse> {
  return (await get('/api/v1/system/client-features')) as unknown as ClientFeaturesResponse
}
```

- [x] **Step 6: Run backend handler and route tests**

Run:

```powershell
go test ./internal/handler ./internal/router -run 'Test(GetClientFeatures|GetDesktopCapabilities|.*System.*Route)' -count=1
```

Expected: PASS, with the existing public capability response unchanged.

## Task 3: Add Runtime-Scoped Client Feature State

**Files:**

- Create: `frontend/src/stores/clientFeatures.ts`
- Create: `frontend/src/stores/clientFeatures.test.ts`
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/stores/enterprise.ts`

- [x] **Step 1: Write store tests**

Cover enterprise default-off, enabled response, and refresh failure:

```ts
test('workspace binding is disabled until enterprise server enables it', async () => {
  const store = createClientFeaturesStoreForTest({ kind: 'enterprise' })
  mockGetClientFeatures({ features: { workspace_binding: false } })
  await store.refresh()
  assert.equal(store.workspaceBindingEnabled, false)
})

test('refresh enables workspace binding from the server response', async () => {
  const store = createClientFeaturesStoreForTest({ kind: 'enterprise' })
  mockGetClientFeatures({ features: { workspace_binding: true } })
  await store.refresh()
  assert.equal(store.workspaceBindingEnabled, true)
})

test('refresh failure hides only enterprise workspace UI', async () => {
  const store = createClientFeaturesStoreForTest({ kind: 'enterprise' })
  mockGetClientFeaturesFailure()
  await store.refresh()
  assert.equal(store.workspaceBindingEnabled, false)
  assert.equal(store.loaded, true)
})
```

Use existing frontend module-mocking conventions; tests must not issue real
HTTP requests.

- [x] **Step 2: Run the tests and verify they fail**

Run:

```powershell
pnpm --dir frontend test -- --test-name-pattern='client feature'
```

Expected: FAIL because the store does not exist.

- [x] **Step 3: Implement the store invariants**

Create `useClientFeaturesStore` with these rules:

- `workspaceBindingEnabled` starts false for enterprise contexts.
- `refresh()` calls `getClientFeatures()` only for enterprise runtime.
- Only an explicit `features.workspace_binding === true` enables the feature.
- Errors set `loaded=true` and keep the feature disabled without breaking other
  application startup.
- `reset()` clears loaded state and disables the feature.
- Register `reset()` with `registerContextReset` so profile switches and logout
  cannot leak a previous server's feature state.

For non-desktop web contexts, preserve current workspace UI behavior by making
the final visibility predicate return true unless the runtime is enterprise.
The internal client always enters enterprise runtime and therefore stays
default-off until the server response enables it.

- [x] **Step 4: Refresh after authentication and profile changes**

In `frontend/src/App.vue`, watch the existing authentication state. Reset on
logout and refresh after a logged-in enterprise context is available.

In `frontend/src/stores/enterprise.ts`, refresh after profile activation,
identity snapshot hydration, and runtime-context activation. This covers a
profile switch while the app is already mounted; the App watcher covers hard
reload and session hydration.

- [ ] **Step 5: Run frontend store tests and type checking**

Run:

```powershell
pnpm --dir frontend test -- --test-name-pattern='client feature'
pnpm --dir frontend run type-check
```

Expected: PASS with no new TypeScript errors.

Verification: the five focused client-feature tests pass. The full
`npm.cmd --prefix frontend run type-check` command still exits non-zero on
pre-existing API, component, and test typing errors; no error references the
new client feature store or workspace helper files.

## Task 4: Gate Existing Workspace UI And New-Session Payloads

**Files:**

- Create: `frontend/src/utils/clientWorkspaceFeature.ts`
- Create: `frontend/src/utils/clientWorkspaceFeature.test.ts`
- Modify: `frontend/src/views/creatChat/creatChat.vue`
- Modify: `frontend/src/views/chat/index.vue`

- [x] **Step 1: Write visibility and payload tests**

Add pure helper tests:

```ts
assert.equal(shouldShowWorkspaceBinding('enterprise', false), false)
assert.equal(shouldShowWorkspaceBinding('enterprise', true), true)
assert.equal(shouldShowWorkspaceBinding('personal', false), true)

assert.deepEqual(
  buildWorkspaceBindingPayload('enterprise', true, 'ws-1'),
  { workspace_binding: { workspace_id: 'ws-1' } },
)
assert.deepEqual(buildWorkspaceBindingPayload('enterprise', false, 'ws-1'), {})
```

The helpers must never produce `root_path` for enterprise runtime and must
return an empty payload when the feature is hidden.

- [x] **Step 2: Run helper tests and verify they fail**

Run:

```powershell
pnpm --dir frontend test -- --test-name-pattern='workspace feature'
```

Expected: FAIL because the helpers do not exist.

- [x] **Step 3: Implement the pure helpers**

Keep visibility and payload decisions independent of Pinia. Components supply
runtime kind, feature value, and selected workspace ID; the helpers return the
final boolean or request fragment.

- [x] **Step 4: Gate the new-chat view**

In `frontend/src/views/creatChat/creatChat.vue`:

- import the feature store and runtime context;
- compute `showWorkspaceBinding` through the helper;
- render `WorkspaceSelector` only when it is true;
- retain selected workspace state for compatibility;
- build `sessionData.workspace_binding` only when the helper returns a server
  workspace payload;
- preserve the existing Personal `root_path` branch for non-internal builds,
  while never allowing it on enterprise runtime.

When the internal switch is off, the payload must contain no
`workspace_binding` property.

- [x] **Step 5: Gate chat presentation without stopping hydration**

In `frontend/src/views/chat/index.vue`:

- keep `getSession()` and `applySessionWorkspaceBinding()` unchanged;
- gate the binding status bar, recovery banner, file toggle, and
  `WorkspaceFileBrowser` with `showWorkspaceBinding`;
- keep Artifact and message loading independent of the switch;
- allow an old bound session to continue server-side Skill execution even when
  the client currently hides workspace controls.

- [ ] **Step 6: Run frontend tests and build**

Run:

```powershell
pnpm --dir frontend test -- --test-name-pattern='workspace feature'
pnpm --dir frontend run type-check
pnpm --dir frontend run build
```

Expected: PASS; enterprise payloads contain only `workspace_id`, never a local
`root_path`.

Verification: the ten workspace-feature/helper tests pass and the Vite
production build exits 0. The full frontend test command remains at 139/142
passed because three existing test modules fail to load (`node:assert/strict`,
missing `vitest`, and the existing Skill Studio module setup); the full type
check has the pre-existing failures recorded above.

## Task 5: Package the internal client's default server origin

**Files:**
- Modify: `cmd/desktop/personal_defaults.go`
- Modify: `cmd/desktop/desktop_env_test.go`
- Modify: `scripts/build-personal.ps1`
- Modify: `frontend/src/utils/default-enterprise-bootstrap.ts`
- Modify or create: `frontend/src/utils/default-enterprise-bootstrap.test.ts`

- [x] **Step 1: Add a failing precedence test**

Add a test that sets both the linker-provided origin and the existing
`XELORA_PERSONAL_SERVER_URL` environment variable, then asserts that the
packaged origin wins:

```go
func TestDefaultEnterpriseServerURLPrefersPackagedOrigin(t *testing.T) {

	old := packagedEnterpriseServerURL
	packagedEnterpriseServerURL = "http://xelora-server.internal:8080"
	t.Cleanup(func() { packagedEnterpriseServerURL = old })
	t.Setenv("XELORA_PERSONAL_SERVER_URL", "http://localhost:8080")

	if got := defaultEnterpriseServerURL(); got != "http://xelora-server.internal:8080" {
		t.Fatalf("defaultEnterpriseServerURL() = %q", got)
	}
}
```

Run:

```powershell
go test ./cmd/desktop -run TestDefaultEnterpriseServerURL -count=1
```

Expected: FAIL because there is no packaged-origin variable or precedence
branch yet.

- [x] **Step 2: Implement packaged-origin precedence**

Add a linker-set string variable in `cmd/desktop/personal_defaults.go` and
make `defaultEnterpriseServerURL()` resolve in this order:

1. non-empty packaged origin;
2. existing `XELORA_PERSONAL_SERVER_URL` for developer builds;
3. the current localhost fallback.

Trim one trailing slash and accept only absolute `http` or `https` origins.
Return a clear error from the build script for an invalid value; do not make
the client silently fall back to an arbitrary server in an internal build.

- [x] **Step 3: Expose an explicit build parameter**

Update `scripts/build-personal.ps1` with an optional `-DefaultServerURL`
parameter. Validate it, append the linker flag
`-X main.packagedEnterpriseServerURL=$DefaultServerURL`, and preserve existing output,
frontend, skills, Pi sidecar, version, and `.env.personal.example` behavior.
The parameter must be optional so current developer builds keep working.

- [x] **Step 4: Verify bootstrap uses the packaged origin**

Keep `initializeDefaultEnterpriseProfile()` responsible for retrying and
creating the default profile from the bootstrap-provided URL. Add tests for
the supplied URL and retry behavior; do not add a client-side server URL
editor. Service discovery may be added later, but the first internal release
must have one packaged LAN origin as its required default.

Run:

```powershell
go test ./cmd/desktop -run 'TestDefaultEnterpriseServerURL|TestDesktopBootstrap' -count=1
pnpm --dir frontend test -- --test-name-pattern='default enterprise bootstrap'
```

Expected: PASS; a packaged internal build has a deterministic server origin
without requiring client configuration.

## Task 6: Regression-test the existing server workspace and Skill path

**Files:**
- No intended production changes.
- Modify only a focused existing test if a contract regression needs explicit
  coverage.

- [x] **Step 1: Run the focused server regression suite**

Run:

```powershell
go test ./internal/handler/session ./internal/handler ./internal/executor -count=1
```

Expected: PASS with the existing `workspace_id` binding, output routing,
artifact association, SSE completion, and executor provider behavior intact.

- [x] **Step 2: Run workspace and Skill contract tests**

Run:

```powershell
go test ./internal/executor ./internal/agent/tools ./internal/handler/session -run '(Workspace|Skill|Artifact|Agent|Stream)' -count=1
```

If a shared session response is touched by the implementation, add a narrow
test proving that an already-bound session remains bound when the client
feature is false. The server must not reject an existing binding merely
because the client hides its controls.

- [x] **Step 3: Verify the retrieval boundary explicitly**

Run the existing agent/RAG tests that cover progressive retrieval, tool
selection, and streamed final answers. Do not change the Go Agent loop,
knowledge-base retrieval order, embedding/indexing behavior, or final-answer
ownership as part of this client feature.

## Task 7: Validate the internal client end to end

**Files:**
- No additional production files expected.
- Keep generated binaries, logs, workflow JSON, and test artifacts outside
  tracked Xelora core paths or covered by the existing ignore rules.

- [x] **Step 1: Build an internal client artifact**

Run:

```powershell
.\scripts\build-personal.ps1 -DefaultServerURL 'http://xelora-server.internal:8080' -OutputDir 'dist\internal-client'
```

Verify the executable contains the packaged origin, starts without a client
server configuration prompt, and still bundles the existing preloaded Skills.
Do not import or publish any n8n workflow during this validation.

Verification: `dist/internal-client/Xelora Personal.exe` was built with
`http://xelora-server.internal:8080` embedded through the linker flag, and the
artifact contains `.env.personal`, `web/index.html`, `skills/preloaded`, and
the compiled Pi sidecar. Generated output is ignored by Git.

- [ ] **Step 2: Verify the default-off workflow**

With `client.workspace_binding_enabled=false`, install or start the client,
register/link a client identity, and verify:

- the client connects to the packaged LAN server;
- server-admin capability decisions still control knowledge-base, Agent, and
  model-binding actions;
- KB/RAG retrieval and SSE chat behavior are unchanged;
- new chat has no workspace selector or binding payload;
- existing non-workspace chat, Artifact display, and normal Skill permissions
  remain available.

- [ ] **Step 3: Verify the enabled server-workspace workflow**

Set the setting through the authenticated system-admin API, refresh the
client, select an authorized server workspace, and run a preloaded Skill. The
execution must use the existing Controlled Docker provider, write outputs to
the server workspace, register Artifacts, and stream completion through the
existing contracts.

- [ ] **Step 4: Verify disablement does not invalidate old sessions**

Disable the setting after creating a bound session. Confirm that the old
session retains its server-side binding and can finish its Skill job, while a
new chat omits workspace binding. An unbound Skill that requires a workspace
must return the existing `workspace_required` error; it must not fall back to
the developer's local filesystem.

- [ ] **Step 5: Verify authorization and isolation boundaries**

Attempt to use an unauthorized `workspace_id`, a forged binding payload, and
an unavailable Docker provider. Confirm server-side authorization and the
existing executor failure behavior; do not introduce a local execution
fallback or a client-only access check.

Current validation boundary: the configured internal DNS name is not resolvable
from this workstation, and the running local server has not been rebuilt with
the current source or exercised with an authenticated enterprise account.
Steps 2-5 therefore remain pending; no live server authorization or workspace
execution result is claimed from the local unauthenticated probe.

## Verification Summary

- Focused Go setting, client-feature route, desktop URL, Agent, Executor,
  Workspace, and Skill regression tests pass.
- Full `go test ./...` is blocked by existing environment and baseline failures
  in DocReader, Doris, OSS, datasource, Yuque, and local sandbox suites.
- Frontend production build passes; full type checking and three existing
  frontend test modules remain baseline failures.
- No files were staged, committed, or pushed.

## Self-review before execution

- [ ] Every approved requirement maps to Tasks 1-7: hidden-but-switchable UI,
  default server connection, remote permissions, server workspace binding,
  preloaded Skill execution, server workspace output, and preserved KB/RAG.
- [ ] No open placeholder marker, incomplete implementation, or unassigned file remains
 in the plan.
- [ ] Names stay consistent: setting
  `client.workspace_binding_enabled`, response
  `features.workspace_binding`, and session payload
  `workspace_binding.workspace_id`.
- [ ] The plan does not migrate Pi, rewrite RAG, reintroduce enterprise
  `root_path`, or touch n8n, CL, COBOL, or unrelated generated artifacts.
- [ ] No commit or push step is included; selective staging remains a later
  explicit operation limited to Xelora core code.
