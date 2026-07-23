# Xelora Personal Enterprise Unified Runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Xelora Personal use the same server-owned identity, tenants, resources, sessions, sharing rules, and RBAC behavior as the web client while preserving an independent personal offline mode.

**Architecture:** The shared Vue application selects a runtime context. Personal context continues to call the embedded local API; enterprise context calls the same remote /api/v1 contract through a loopback-only desktop transport gateway that owns credentials, token refresh, TLS policy, streaming, and target validation. Enterprise business data is never copied into local SQLite.

**Tech Stack:** Go 1.26, Gin, GORM, Windows Credential Manager APIs, Wails v2, Vue 3.5, Pinia 3, Axios, @microsoft/fetch-event-source, Node test runner, PowerShell build scripts.

---

## Execution Preconditions

The current worktree contains overlapping uncommitted changes in cmd/desktop,
frontend/src/stores/auth.ts, frontend/src/views/chat/index.vue,
internal/container/container.go, internal/router/router.go, and the legacy
enterprise implementation. Before Task 1:

- [ ] Record `git status --short` and `git diff --name-only`.
- [ ] Confirm commit `aeb82bdd` and the approved design are present.
- [ ] Execute in the current worktree unless the user first approves a
  selective baseline commit containing the overlapping changes. A clean
  worktree created only from HEAD would omit behavior this plan was designed
  against.
- [ ] Never stage unrelated modified or untracked files. Every commit command
  below names its files explicitly.

Run:

~~~powershell
git status --short
git rev-parse --short HEAD
git show --stat --oneline aeb82bdd
~~~

Expected: the design commit is present and the dirty-file inventory is saved
in the task notes.

## File Map

New backend units:

- internal/desktopremote/profile.go: server profile model and URL policy.
- internal/desktopremote/profile_store.go: profile persistence and legacy
  metadata migration.
- internal/desktopremote/credential_store.go: credential-store interface.
- internal/desktopremote/credential_store_windows.go: Windows Credential
  Manager implementation.
- internal/desktopremote/credential_store_other.go: non-Windows memory-only
  development implementation.
- internal/desktopremote/session.go: in-memory access tokens and token-free
  identity snapshots.
- internal/desktopremote/gateway.go: target-locked HTTP/SSE/upload/download
  forwarding.
- internal/desktopremote/auth.go: login, refresh, logout, and session restore.
- internal/desktopremote/manager.go: profile, session, capability, and gateway
  orchestration.
- internal/handler/desktop_remote.go: loopback desktop API surface.
- internal/handler/system_capabilities.go: public API contract handshake.

New frontend units:

- frontend/src/stores/runtimeContext.ts: active personal/server/user/tenant
  context and generation.
- frontend/src/utils/api-context.ts: context-derived local API prefix.
- frontend/src/api/desktop-remote/index.ts: profiles, capabilities, session, and
  context APIs.
- frontend/src/utils/context-reset.ts: abort and store-reset registry.
- frontend/src/components/RuntimeContextSelector.vue: personal/server switcher.

Existing files changed together:

- internal/router/router.go and internal/container/container.go wire the new
  public capability and loopback gateway surfaces.
- cmd/desktop/app.go and cmd/desktop/main.go inject the per-process desktop
  gateway secret and register deep-link handling.
- frontend/src/utils/request.ts and frontend/src/api/chat/streame.ts route all
  request types through the active context.
- frontend/src/stores/auth.ts, frontend/src/router/index.ts,
  frontend/src/views/auth/Login.vue, and frontend/src/App.vue support token-free
  enterprise desktop sessions while preserving local token auth.
- frontend/src/components/AgentSelector.vue,
  frontend/src/components/KnowledgeBaseSelector.vue,
  frontend/src/views/chat/index.vue, and frontend/src/stores/settings.ts remove
  enterprise resource IDs and local/remote session mapping.
- frontend/src/components/EnterpriseServerManager.vue becomes a server-profile
  manager with login state, TLS policy, and capability status.
- scripts/installer-personal.nsi and cmd/desktop/main.go implement xelora://
  protocol registration and dispatch.

## Phase 1: Contract And Desktop Foundation

### Task 1: Public Server Capability Contract

**Files:**

- Create: internal/handler/system_capabilities.go
- Create: internal/handler/system_capabilities_test.go
- Modify: internal/router/router.go
- Modify: docs/swagger.yaml

- [ ] **Step 1: Write the failing capability handler test**

~~~go
func TestGetDesktopCapabilitiesIsPublicAndStable(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    r.GET("/api/v1/system/capabilities", GetSystemCapabilities)

    w := httptest.NewRecorder()
    r.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
        "/api/v1/system/capabilities", nil))

    if w.Code != http.StatusOK {
        t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
    }
    var got SystemCapabilitiesResponse
    if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
        t.Fatal(err)
    }
    if got.APIContractMajor != 1 || got.APIContractMinor < 0 {
        t.Fatalf("unexpected contract: %+v", got)
    }
    required := map[string]bool{
        "tenant_rbac": false, "organizations": false,
        "shared_resources": false, "sse_chat": false,
    }
    for _, feature := range got.Features {
        if _, ok := required[feature]; ok {
            required[feature] = true
        }
    }
    for feature, present := range required {
        if !present {
            t.Fatalf("missing feature %q", feature)
        }
    }
}
~~~

- [ ] **Step 2: Run the test and verify the symbol is missing**

Run:

~~~powershell
go test ./internal/handler -run TestGetDesktopCapabilitiesIsPublicAndStable -count=1
~~~

Expected: FAIL because SystemCapabilitiesResponse and
GetSystemCapabilities do not exist.

- [ ] **Step 3: Add the immutable capability response**

~~~go
package handler

import (
    "net/http"
    "runtime/debug"

    "github.com/gin-gonic/gin"
)

const (
    DesktopAPIContractMajor = 1
    DesktopAPIContractMinor = 0
)

type SystemCapabilitiesResponse struct {
    APIContractMajor int      `json:"api_contract_major"`
    APIContractMinor int      `json:"api_contract_minor"`
    ServerVersion    string   `json:"server_version"`
    Features         []string `json:"features"`
}

func GetSystemCapabilities(c *gin.Context) {
    version := "dev"
    if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
        version = info.Main.Version
    }
    c.JSON(http.StatusOK, SystemCapabilitiesResponse{
        APIContractMajor: DesktopAPIContractMajor,
        APIContractMinor: DesktopAPIContractMinor,
        ServerVersion: version,
        Features: []string{
            "tenant_rbac", "organizations", "shared_resources", "sse_chat",
        },
    })
}
~~~

Register `GET /api/v1/system/capabilities` before
`r.Use(middleware.Auth(...))`. Do not add it to the authenticated
RegisterSystemRoutes group.

- [ ] **Step 4: Verify public routing and handler tests**

Run:

~~~powershell
go test ./internal/handler ./internal/router -run "Capabilities|Router" -count=1
~~~

Expected: PASS, and an unauthenticated request returns 200.

- [ ] **Step 5: Document the response and commit**

Add the exact response fields and public/no-user-data semantics to
docs/swagger.yaml.

~~~powershell
git add -- internal/handler/system_capabilities.go internal/handler/system_capabilities_test.go internal/router/router.go docs/swagger.yaml
git commit -m "feat: expose desktop capability contract"
~~~

### Task 2: Server Profiles And URL Security

**Files:**

- Create: internal/desktopremote/profile.go
- Create: internal/desktopremote/profile_store.go
- Create: internal/desktopremote/profile_test.go
- Modify: internal/container/container.go

- [ ] **Step 1: Write failing normalization and target-lock tests**

~~~go
func TestNormalizeServerOrigin(t *testing.T) {
    tests := []struct {
        raw, want string
        insecure bool
        wantErr bool
    }{
        {"https://xelora.example.com/", "https://xelora.example.com", false, false},
        {"http://127.0.0.1:8080", "http://127.0.0.1:8080", false, false},
        {"http://192.168.1.10:8080", "http://192.168.1.10:8080", true, false},
        {"https://user:pass@example.com", "", false, true},
        {"file:///c:/data", "", false, true},
    }
    for _, tt := range tests {
        got, err := NormalizeServerOrigin(tt.raw, tt.insecure)
        if (err != nil) != tt.wantErr || got != tt.want {
            t.Fatalf("%q => %q, %v", tt.raw, got, err)
        }
    }
}

func TestResolveTargetCannotChangeOrigin(t *testing.T) {
    profile := ServerProfile{BaseURL: "https://xelora.example.com"}
    if _, err := profile.ResolveTarget("//attacker.example/api/v1"); err == nil {
        t.Fatal("scheme-relative target must be rejected")
    }
}
~~~

- [ ] **Step 2: Run the package test and verify failure**

~~~powershell
go test ./internal/desktopremote -run "Normalize|ResolveTarget" -count=1
~~~

Expected: FAIL because the package does not exist.

- [ ] **Step 3: Implement the focused profile model**

~~~go
type ServerProfile struct {
    ID                            string
    Name                          string
    BaseURL                       string
    AllowInsecureTransport        bool
    TrustedCertificateFingerprint string
    LastUserID                    string
    LastTenantID                  uint64
    CreatedAt                     time.Time
    UpdatedAt                     time.Time
}

func (ServerProfile) TableName() string { return "desktop_server_profiles" }

func NormalizeServerOrigin(raw string, allowInsecure bool) (string, error) {
    u, err := url.Parse(strings.TrimSpace(raw))
    if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
        return "", errors.New("server address must be an http(s) origin")
    }
    u.Path = strings.TrimRight(u.Path, "/")
    if u.Path != "" {
        return "", errors.New("server address must not contain a path")
    }
    host := strings.ToLower(u.Hostname())
    local := host == "localhost" || net.ParseIP(host) != nil &&
        net.ParseIP(host).IsLoopback()
    if u.Scheme != "https" && !(u.Scheme == "http" && (local || allowInsecure)) {
        return "", errors.New("HTTPS is required")
    }
    return strings.TrimRight(u.String(), "/"), nil
}
~~~

Implement Create, List, Get, Update, and Delete in ProfileStore. Normalize the
origin before every write and use a new desktop_server_profiles table so legacy
secrets are not copied accidentally.

- [ ] **Step 4: Add SQLite store tests**

Use `gorm.Open(sqlite.Open("file::memory:?cache=shared"))`, call AutoMigrate,
create two profiles, and assert ordered listing, immutable ID, normalized URL,
and delete behavior.

- [ ] **Step 5: Run tests and commit**

~~~powershell
go test ./internal/desktopremote -run "Profile|Normalize|ResolveTarget" -count=1
git add -- internal/desktopremote/profile.go internal/desktopremote/profile_store.go internal/desktopremote/profile_test.go internal/container/container.go
git commit -m "feat: add secure desktop server profiles"
~~~

### Task 3: Credential Store Boundary

**Files:**

- Create: internal/desktopremote/credential_store.go
- Create: internal/desktopremote/credential_store_windows.go
- Create: internal/desktopremote/credential_store_other.go
- Create: internal/desktopremote/credential_store_test.go
- Modify: go.mod
- Modify: go.sum

- [ ] **Step 1: Write the contract test against a fake store**

~~~go
type CredentialStore interface {
    PutRefreshToken(ctx context.Context, profileID, userID, token string) error
    GetRefreshToken(ctx context.Context, profileID, userID string) (string, error)
    DeleteRefreshToken(ctx context.Context, profileID, userID string) error
}

func credentialTarget(profileID, userID string) string {
    return "Xelora Personal/enterprise/" + profileID + "/" + userID
}

func TestCredentialTargetIsStableAndScoped(t *testing.T) {
    a := credentialTarget("server-a", "user-1")
    b := credentialTarget("server-b", "user-1")
    if a == b || a != "Xelora Personal/enterprise/server-a/user-1" {
        t.Fatalf("unexpected targets %q %q", a, b)
    }
}
~~~

- [ ] **Step 2: Run the focused test and verify failure**

~~~powershell
go test ./internal/desktopremote -run Credential -count=1
~~~

Expected: FAIL until the interface and target helper are added.

- [ ] **Step 3: Implement Windows Credential Manager storage**

Use `golang.org/x/sys/windows` for UTF-16 conversion and direct lazy-DLL
calls to `CredWriteW`, `CredReadW`, `CredDeleteW`, and `CredFree`.
Write a generic credential with:

~~~go
blob := []byte(token)
defer zeroBytes(blob)
credential := windowsCredential{
    Type:       credTypeGeneric,
    TargetName: windows.StringToUTF16Ptr(credentialTarget(profileID, userID)),
    CredentialBlobSize: uint32(len(blob)),
    CredentialBlob:     &blob[0],
    Persist:            credPersistLocalMachine,
    UserName:           windows.StringToUTF16Ptr(userID),
}
~~~

Return a typed ErrCredentialNotFound for ERROR_NOT_FOUND. Zero temporary token
byte slices after each native call. The non-Windows implementation is
memory-only and exists for tests/development; it must never write plaintext to
disk.

- [ ] **Step 4: Add Windows build and fake-store behavior checks**

~~~powershell
go test ./internal/desktopremote -run Credential -count=1
go test -c -o "$env:TEMP\xelora-desktopremote.test.exe" ./internal/desktopremote
~~~

Expected: PASS and the Windows test binary links.

- [ ] **Step 5: Commit**

~~~powershell
git add -- internal/desktopremote/credential_store.go internal/desktopremote/credential_store_windows.go internal/desktopremote/credential_store_other.go internal/desktopremote/credential_store_test.go go.mod go.sum
git commit -m "feat: store enterprise refresh tokens in Windows credentials"
~~~

### Task 4: Token-Free Desktop Sessions

**Files:**

- Create: internal/desktopremote/session.go
- Create: internal/desktopremote/session_test.go
- Create: internal/desktopremote/auth.go
- Create: internal/desktopremote/auth_test.go

- [ ] **Step 1: Write failing session isolation and redaction tests**

~~~go
func TestSessionSnapshotNeverContainsTokens(t *testing.T) {
    sessions := NewSessionStore()
    sessions.Set("server-a", &Session{
        UserID: "u1", AccessToken: "access", RefreshTokenOwner: "u1",
        Snapshot: IdentitySnapshot{Authenticated: true, UserID: "u1"},
    })
    raw, err := json.Marshal(sessions.Snapshot("server-a"))
    if err != nil {
        t.Fatal(err)
    }
    if bytes.Contains(raw, []byte("access")) {
        t.Fatalf("snapshot leaked token: %s", raw)
    }
    if sessions.Snapshot("server-b").Authenticated {
        t.Fatal("sessions must be scoped by profile")
    }
}
~~~

- [ ] **Step 2: Run and verify failure**

~~~powershell
go test ./internal/desktopremote -run "Session|Login|Refresh" -count=1
~~~

- [ ] **Step 3: Implement session state and login parsing**

~~~go
type IdentitySnapshot struct {
    Authenticated bool               `json:"authenticated"`
    User          json.RawMessage    `json:"user,omitempty"`
    Tenant        json.RawMessage    `json:"tenant,omitempty"`
    Memberships   []json.RawMessage  `json:"memberships,omitempty"`
    ExpiresAt     time.Time          `json:"expires_at,omitempty"`
}

type Session struct {
    UserID          string
    AccessToken     string
    AccessExpiresAt time.Time
    Snapshot        IdentitySnapshot
}
~~~

Login forwards POST /api/v1/auth/login, decodes token and refresh_token,
persists only the refresh token in CredentialStore, keeps the access token in
SessionStore, and returns IdentitySnapshot. Refresh POSTs the stored refresh
token, rotates it in CredentialStore, and updates SessionStore.

- [ ] **Step 4: Test upstream login, refresh rotation, logout, and redaction**

Use httptest.Server with counters. Assert:

- login response tokens never appear in the returned JSON;
- refresh runs once after an expired access token;
- rotated refresh token replaces the old credential;
- logout removes memory and Credential Manager state;
- profiles never share sessions.

- [ ] **Step 5: Run and commit**

~~~powershell
go test ./internal/desktopremote -run "Session|Login|Refresh|Logout" -count=1
git add -- internal/desktopremote/session.go internal/desktopremote/session_test.go internal/desktopremote/auth.go internal/desktopremote/auth_test.go
git commit -m "feat: add token-free enterprise desktop sessions"
~~~

### Task 5: Generic Desktop Transport Gateway

**Files:**

- Create: internal/desktopremote/gateway.go
- Create: internal/desktopremote/gateway_test.go

- [ ] **Step 1: Write failing forwarding security tests**

~~~go
func TestGatewayLocksTargetAndRebuildsHeaders(t *testing.T) {
    upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/api/v1/agents" {
            t.Fatalf("path=%s", r.URL.Path)
        }
        if r.Header.Get("Authorization") != "Bearer access-token" {
            t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
        }
        if r.Header.Get("X-Tenant-ID") != "42" {
            t.Fatalf("tenant=%q", r.Header.Get("X-Tenant-ID"))
        }
        if r.Header.Get("X-Forwarded-Host") != "" {
            t.Fatal("forwarding header leaked")
        }
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"ok":true}`))
    }))
    defer upstream.Close()
    // Build profile/session fakes, invoke gateway, assert status and body.
}
~~~

Also add tests for a missing desktop secret, unsaved profile, scheme-relative
path, cross-origin redirect, caller-supplied Authorization header, changed
pinned certificate, and DNS resolution outside the IP set approved at profile
activation.

- [ ] **Step 2: Run and verify failure**

~~~powershell
go test ./internal/desktopremote -run Gateway -count=1
~~~

- [ ] **Step 3: Implement one generic forwarding path**

~~~go
var forwardedHeaders = map[string]bool{
    "Accept": true, "Accept-Language": true, "Content-Type": true,
    "Content-Range": true, "Range": true, "X-Request-ID": true,
    "X-Tenant-ID": true, "Idempotency-Key": true,
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    profileID, remotePath, err := parseGatewayPath(r.URL.Path)
    if err != nil {
        writeGatewayError(w, http.StatusBadRequest, "invalid_remote_path")
        return
    }
    profile, err := g.profiles.Get(r.Context(), profileID)
    if err != nil {
        writeGatewayError(w, http.StatusNotFound, "server_profile_not_found")
        return
    }
    target, err := profile.ResolveTarget(remotePath)
    if err != nil {
        writeGatewayError(w, http.StatusBadRequest, "invalid_remote_target")
        return
    }
    // Create request with original method/body, copy only forwardedHeaders,
    // inject SessionStore access token, execute, and stream headers/body.
}
~~~

Use an http.Client with CheckRedirect returning http.ErrUseLastResponse.
Preserve response status, Content-Type, Content-Disposition, Content-Range,
Accept-Ranges, ETag, Last-Modified, and safe application headers. Flush after
each SSE write and propagate request cancellation.

Manager.Activate resolves the profile hostname and records the resulting IP
set in process memory. The transport DialContext resolves again and permits
only an address in that approved set; a changed set returns
ErrServerAddressChanged and requires explicit reactivation. For a pinned
self-signed certificate, set InsecureSkipVerify only together with a
VerifyConnection callback that SHA-256 hashes cert.Raw and compares it with
TrustedCertificateFingerprint using subtle.ConstantTimeCompare. Without a
pin, use normal Go hostname and trust-chain verification.

- [ ] **Step 4: Test JSON, multipart, range download, and SSE**

Add table tests that send:

- JSON POST and verify byte-identical body;
- multipart upload and verify boundary/content;
- Range request and verify 206 plus Content-Range;
- two delayed SSE events and verify the first is observable before completion.
- TLS server with the approved fingerprint succeeds and a changed certificate
  fails before any HTTP body is sent;
- activation IP set A followed by resolver result B fails closed.

- [ ] **Step 5: Run race tests and commit**

~~~powershell
go test -race ./internal/desktopremote -run Gateway -count=1
git add -- internal/desktopremote/gateway.go internal/desktopremote/gateway_test.go
git commit -m "feat: add target-locked desktop transport gateway"
~~~

### Task 6: Gateway Auth Retry And Capability Check

**Files:**

- Create: internal/desktopremote/manager.go
- Create: internal/desktopremote/manager_test.go
- Modify: internal/desktopremote/gateway.go

- [ ] **Step 1: Write failing one-refresh and compatibility tests**

Create an upstream that returns 401 once, accepts the refreshed token once, and
records request count. Assert GET is replayed exactly once, POST without an
Idempotency-Key returns the original 401, and POST with Idempotency-Key may be
replayed once.

Add:

~~~go
func TestCapabilityMajorMismatchBlocksActivation(t *testing.T) {
    err := ValidateCapabilities(SystemCapabilities{
        APIContractMajor: 2,
        APIContractMinor: 0,
    })
    if !errors.Is(err, ErrIncompatibleAPIContract) {
        t.Fatalf("err=%v", err)
    }
}
~~~

- [ ] **Step 2: Run and verify failure**

~~~powershell
go test ./internal/desktopremote -run "Capability|Retry" -count=1
~~~

- [ ] **Step 3: Implement Manager.Activate and replay policy**

Define the shared desktopremote response type before Manager:

~~~go
type SystemCapabilities struct {
    APIContractMajor int
    APIContractMinor int
    ServerVersion    string
    Features         []string
}
~~~

Manager.Activate fetches /api/v1/system/capabilities without credentials,
requires major version 1, records minor/features, then restores the last
session if a credential exists. Gateway retry eligibility is:

~~~go
func replayable(r *http.Request) bool {
    switch r.Method {
    case http.MethodGet, http.MethodHead, http.MethodOptions:
        return true
    default:
        return r.Header.Get("Idempotency-Key") != ""
    }
}
~~~

- [ ] **Step 4: Run package tests**

~~~powershell
go test -race ./internal/desktopremote -count=1
~~~

Expected: PASS with no data race in concurrent refresh queuing.

- [ ] **Step 5: Commit**

~~~powershell
git add -- internal/desktopremote/manager.go internal/desktopremote/manager_test.go internal/desktopremote/gateway.go
git commit -m "feat: coordinate enterprise capability and auth sessions"
~~~

## Phase 2: Backend Wiring And Runtime Context

### Task 7: Loopback Desktop API And Dependency Injection

**Files:**

- Create: internal/handler/desktop_remote.go
- Create: internal/handler/desktop_remote_test.go
- Modify: internal/router/router.go
- Modify: internal/container/container.go
- Modify: cmd/desktop/app.go
- Modify: cmd/desktop/main.go
- Modify generated bindings: frontend/src/wailsjs/go/main/App.js
- Modify generated bindings: frontend/src/wailsjs/go/main/App.d.ts

- [ ] **Step 1: Write failing route tests**

Build a Gin router with the desktop handler and assert:

- POST /desktop/remote/profiles creates a normalized profile;
- POST /desktop/remote/profiles/:id/activate returns capabilities;
- POST /desktop/remote/profiles/:id/login returns a token-free snapshot;
- GET /desktop/remote/profiles/:id/session restores a snapshot;
- ANY /desktop/remote/profiles/:id/api/v1/*path reaches Gateway;
- every route rejects a missing or wrong X-Xelora-Desktop-Session value.

- [ ] **Step 2: Run and verify failure**

~~~powershell
go test ./internal/handler -run DesktopRemote -count=1
~~~

- [ ] **Step 3: Implement one handler around Manager**

~~~go
type DesktopRemoteHandler struct {
    manager *desktopremote.Manager
    secret  string
}

func (h *DesktopRemoteHandler) RequireDesktopSession() gin.HandlerFunc {
    return func(c *gin.Context) {
        if subtle.ConstantTimeCompare(
            []byte(c.GetHeader("X-Xelora-Desktop-Session")),
            []byte(h.secret),
        ) != 1 {
            c.AbortWithStatusJSON(http.StatusUnauthorized,
                gin.H{"error": "desktop_session_required"})
            return
        }
        c.Next()
    }
}
~~~

Register /desktop/remote before global server Auth middleware, but only when
handler.Edition == "personal". Inject Manager and handler through the existing
dig container.

- [ ] **Step 4: Expose bootstrap state through the Wails binding before Vue mounts**

DesktopRemoteHandler generates 32 random bytes in its constructor. After the
container is built, main.go invokes the handler from dig and copies its secret
into App. App exposes a token-free server bootstrap except for the local
per-process gateway secret:

~~~go
type DesktopBootstrap struct {
    APIBaseURL string `json:"api_base_url"`
    Session    string `json:"session"`
}

func (a *App) GetDesktopBootstrap() DesktopBootstrap {
    return DesktopBootstrap{
        APIBaseURL: strings.TrimRight(a.backendURL, "/"),
        Session: a.desktopSessionSecret,
    }
}
~~~

Regenerate Wails bindings, then have main.ts await GetDesktopBootstrap before
createApp/mount and assign the result to an in-memory module variable in
api-context.ts. Never emit this value in logs, localStorage, sessionStorage, or
GetAPIBaseURL.

- [ ] **Step 5: Verify backend wiring and commit**

~~~powershell
go test ./internal/desktopremote ./internal/handler ./internal/router -count=1
go test ./cmd/desktop -count=1
go run github.com/wailsapp/wails/v2/cmd/wails generate module
git add -- internal/handler/desktop_remote.go internal/handler/desktop_remote_test.go internal/router/router.go internal/container/container.go cmd/desktop/app.go cmd/desktop/main.go frontend/src/wailsjs/go/main/App.js frontend/src/wailsjs/go/main/App.d.ts
git commit -m "feat: wire desktop remote gateway routes"
~~~

### Task 8: Frontend Runtime Context And Dynamic API Routing

**Files:**

- Create: frontend/src/stores/runtimeContext.ts
- Create: frontend/src/utils/api-context.ts
- Create: frontend/src/utils/api-context.test.ts
- Create: frontend/src/utils/context-reset.ts
- Modify: frontend/src/utils/request.ts
- Modify: frontend/src/api/chat/streame.ts
- Modify: frontend/src/main.ts

- [ ] **Step 1: Write failing pure routing tests**

~~~ts
test('enterprise URLs use the desktop gateway prefix', () => {
  const context = {
    kind: 'enterprise' as const,
    profileId: 'server-a',
    generation: 7,
  }
  assert.equal(
    resolveApiUrl(context, '/api/v1/agents'),
    '/desktop/remote/profiles/server-a/api/v1/agents',
  )
})

test('personal URLs are unchanged', () => {
  assert.equal(
    resolveApiUrl({ kind: 'personal', generation: 1 }, '/api/v1/agents'),
    '/api/v1/agents',
  )
})
~~~

- [ ] **Step 2: Run and verify failure**

~~~powershell
Set-Location frontend
npm test -- src/utils/api-context.test.ts
~~~

- [ ] **Step 3: Implement context and request metadata**

RuntimeContextStore exposes:

~~~ts
type RuntimeContext =
  | { kind: 'personal'; generation: number }
  | {
      kind: 'enterprise'
      profileId: string
      userId: string | null
      tenantId: number | null
      generation: number
    }
~~~

The Axios request interceptor resolves every URL at request time, adds
X-Xelora-Desktop-Session only for enterprise context, adds X-Tenant-ID from
the context, and stamps X-Xelora-Context-Generation. Remove the module-level
constant BASE_URL.

- [ ] **Step 4: Route SSE through the same resolver**

In streame.ts, call resolveApiUrl for fetchEventSource and use one AbortController
registered with context-reset.ts. Drop enterprise_server_id and
X-Enterprise-Server-ID handling.

- [ ] **Step 5: Verify tests, type-check, and commit**

~~~powershell
npm test -- src/utils/api-context.test.ts
npm run type-check
Set-Location ..
git add -- frontend/src/stores/runtimeContext.ts frontend/src/utils/api-context.ts frontend/src/utils/api-context.test.ts frontend/src/utils/context-reset.ts frontend/src/utils/request.ts frontend/src/api/chat/streame.ts frontend/src/main.ts
git commit -m "feat: route frontend requests by runtime context"
~~~

### Task 9: Atomic Context Switching

**Files:**

- Create: frontend/src/stores/runtimeContext.test.ts
- Modify: frontend/src/stores/runtimeContext.ts
- Modify: frontend/src/utils/context-reset.ts
- Modify: frontend/src/stores/auth.ts
- Modify: frontend/src/stores/chatResources.ts
- Modify: frontend/src/stores/editorResources.ts
- Modify: frontend/src/stores/organization.ts
- Modify: frontend/src/stores/settings.ts

- [ ] **Step 1: Write failing generation and cancellation tests**

Test that switchContext:

1. aborts registered requests;
2. increments generation;
3. invokes every registered store reset;
4. activates destination only after reset;
5. rejects a response tagged with the prior generation.

~~~ts
assert.equal(acceptResponse({ generation: 3 }, 4), false)
assert.equal(acceptResponse({ generation: 4 }, 4), true)
~~~

- [ ] **Step 2: Run and verify failure**

~~~powershell
Set-Location frontend
npm test -- src/stores/runtimeContext.test.ts
~~~

- [ ] **Step 3: Implement the reset registry**

~~~ts
const aborters = new Set<AbortController>()
const resets = new Set<() => void | Promise<void>>()

export async function resetActiveContext() {
  for (const controller of aborters) controller.abort('context-switch')
  aborters.clear()
  for (const reset of resets) await reset()
}
~~~

Each listed store exports a focused reset method. Keep personal-only persisted
preferences, but namespace enterprise UI preferences with
enterprise:<profile>:<user>:<tenant>.

- [ ] **Step 4: Run store and type tests**

~~~powershell
npm test -- src/stores/runtimeContext.test.ts
npm run type-check
~~~

- [ ] **Step 5: Commit**

~~~powershell
Set-Location ..
git add -- frontend/src/stores/runtimeContext.ts frontend/src/stores/runtimeContext.test.ts frontend/src/utils/context-reset.ts frontend/src/stores/auth.ts frontend/src/stores/chatResources.ts frontend/src/stores/editorResources.ts frontend/src/stores/organization.ts frontend/src/stores/settings.ts
git commit -m "feat: isolate desktop runtime contexts"
~~~

## Phase 3: Unified Identity And Web Feature Parity

### Task 10: Enterprise Login Without Frontend Tokens

**Files:**

- Create: frontend/src/api/desktop-remote/index.ts
- Create: frontend/src/api/desktop-remote/session.test.ts
- Modify: frontend/src/stores/auth.ts
- Modify: frontend/src/router/index.ts
- Modify: frontend/src/views/auth/Login.vue
- Modify: frontend/src/App.vue
- Modify: frontend/src/utils/request.ts

- [ ] **Step 1: Write failing enterprise-session mapping tests**

~~~ts
test('enterprise snapshot authenticates without a token', () => {
  const state = applyDesktopIdentitySnapshot({
    authenticated: true,
    user: { id: 'u1', username: 'alice', email: 'a@example.com' },
    tenant: { id: 42, name: 'Team' },
    memberships: [{ tenant_id: 42, tenant_name: 'Team', role: 'owner' }],
  })
  assert.equal(state.isLoggedIn, true)
  assert.equal(state.token, '')
  assert.equal(localStorage.getItem('xelora_token'), null)
})
~~~

- [ ] **Step 2: Run and verify failure**

~~~powershell
Set-Location frontend
npm test -- src/api/desktop-remote/session.test.ts
~~~

- [ ] **Step 3: Split local and enterprise auth state**

AuthStore computes isLoggedIn as local token plus user in personal context, or
authenticated desktop snapshot plus user in enterprise context. Add
applyDesktopIdentitySnapshot and clearDesktopIdentity. Do not call setToken or
setRefreshToken for enterprise responses.

- [ ] **Step 4: Adapt login, restore, refresh, and logout flows**

- Login.vue sends enterprise credentials to
  /desktop/remote/profiles/:id/login.
- Router hydration calls /desktop/remote/profiles/:id/session.
- request.ts does not run frontend refresh logic for /desktop/remote paths;
  gateway owns refresh.
- App.vue OIDC hash handling remains personal/web-only until Task 13.
- Logout calls the profile logout endpoint and clears only active enterprise
  state.

- [ ] **Step 5: Verify tokens are absent and commit**

~~~powershell
npm test -- src/api/desktop-remote/session.test.ts
npm run type-check
rg -n "xelora_token|xelora_refresh_token" src/stores/auth.ts src/views/auth/Login.vue src/router/index.ts
~~~

Expected: remaining localStorage token paths are explicitly guarded by personal
context.

~~~powershell
Set-Location ..
git add -- frontend/src/api/desktop-remote/index.ts frontend/src/api/desktop-remote/session.test.ts frontend/src/stores/auth.ts frontend/src/router/index.ts frontend/src/views/auth/Login.vue frontend/src/App.vue frontend/src/utils/request.ts
git commit -m "feat: use server-native login in enterprise context"
~~~

### Task 11: Server And Tenant Context UI

**Files:**

- Create: frontend/src/components/RuntimeContextSelector.vue
- Create: frontend/src/components/runtimeContextMenu.ts
- Create: frontend/src/components/runtimeContextMenu.test.ts
- Modify: frontend/src/components/EnterpriseServerManager.vue
- Modify: frontend/src/components/UserMenu.vue
- Modify: frontend/src/components/TenantSelector.vue
- Modify: frontend/src/components/menu.vue
- Modify: frontend/src/views/settings/Settings.vue
- Modify: frontend/src/i18n/locales/zh-CN.ts
- Modify: frontend/src/i18n/locales/en-US.ts

- [ ] **Step 1: Add a pure context-menu model test**

Test menu items for personal plus two servers, active marker, login-required
status, incompatible status, and last tenant label. Keep the model builder in
RuntimeContextSelector.vue's adjacent runtimeContextMenu.ts if importing a Vue
SFC is not supported by node:test.

- [ ] **Step 2: Run the test and verify failure**

~~~powershell
Set-Location frontend
npm test -- src/components/runtimeContextMenu.test.ts
~~~

- [ ] **Step 3: Replace API-token server form**

EnterpriseServerManager fields become name, base URL, allow insecure transport,
and optional pinned certificate approval. Remove API Token and auto-connect.
Activation performs capability handshake, then shows server-native login.

- [ ] **Step 4: Add runtime and tenant switch surfaces**

RuntimeContextSelector is visible in the desktop shell. After enterprise login,
reuse memberships and TenantSelector. Server switch invokes switchContext;
tenant switch updates RuntimeContextStore tenant ID before reloading.

- [ ] **Step 5: Build and commit**

~~~powershell
npm test -- src/components/runtimeContextMenu.test.ts
npm run type-check
npm run build
Set-Location ..
git add -- frontend/src/components/RuntimeContextSelector.vue frontend/src/components/runtimeContextMenu.ts frontend/src/components/runtimeContextMenu.test.ts frontend/src/components/EnterpriseServerManager.vue frontend/src/components/UserMenu.vue frontend/src/components/TenantSelector.vue frontend/src/components/menu.vue frontend/src/views/settings/Settings.vue frontend/src/i18n/locales/zh-CN.ts frontend/src/i18n/locales/en-US.ts
git commit -m "feat: add personal and enterprise context switching"
~~~

### Task 12: Remove Enterprise Resource Mapping From Active Flow

**Files:**

- Modify: frontend/src/components/AgentSelector.vue
- Modify: frontend/src/components/KnowledgeBaseSelector.vue
- Modify: frontend/src/views/chat/index.vue
- Modify: frontend/src/stores/settings.ts
- Modify: frontend/src/components/ResourceOriginBadge.vue
- Create: frontend/src/utils/enterpriseParity.test.ts
- Delete after migration gate passes: frontend/src/stores/enterprise.ts
- Delete after migration gate passes: frontend/src/api/enterprise/index.ts

- [ ] **Step 1: Write regression tests for native IDs and sessions**

Add pure tests asserting:

- an enterprise-context agent selection stores server ID `agent-7`, not
  `enterprise:server-a:agent-7`;
- a session created in enterprise context uses the returned server session ID;
- agent, KB, session, and organization API paths remain ordinary /api/v1 paths
  and are transformed only by api-context.ts.

- [ ] **Step 2: Run and verify current prefix/mapping behavior fails**

~~~powershell
Set-Location frontend
npm test -- src/utils/enterpriseParity.test.ts
~~~

- [ ] **Step 3: Remove enterprise-only selector branches**

AgentSelector and KnowledgeBaseSelector load the same API/store data in both
web and enterprise desktop contexts. Preserve existing shared-resource badges
that are driven by server sharing metadata; remove badges based only on
origin=enterprise.

- [ ] **Step 4: Remove local/server chat-session mapping**

Delete enterpriseSessionMap, createServerSession, enterprise_server_id, and
the /api/v1/enterprise/chat endpoint selection. Session creation and chat use
/api/v1/sessions and /api/v1/agent-chat/:session_id, which api-context routes.

- [ ] **Step 5: Verify parity and commit**

~~~powershell
npm test -- src/utils/enterpriseParity.test.ts
npm run type-check
npm run build
Set-Location ..
git add -- frontend/src/components/AgentSelector.vue frontend/src/components/KnowledgeBaseSelector.vue frontend/src/views/chat/index.vue frontend/src/stores/settings.ts frontend/src/components/ResourceOriginBadge.vue frontend/src/utils/enterpriseParity.test.ts
git commit -m "refactor: use native server resources in enterprise context"
~~~

Do not delete the legacy enterprise API/store until Task 14 proves migration
and rollback behavior.

## Phase 4: Migration, OIDC, And Deep Links

### Task 13: Legacy Profile Migration And Secret Cleanup

**Files:**

- Create: internal/desktopremote/migration.go
- Create: internal/desktopremote/migration_test.go
- Modify: internal/enterprise/store.go
- Create: frontend/src/utils/enterpriseLegacyAbsence.test.mjs
- Modify: internal/container/container.go

- [ ] **Step 1: Write migration tests with legacy secret fixtures**

Seed enterprise_servers with name, base_url, API token, linked password, JWT,
refresh token, linked IDs, and cached resources. Assert migration copies only
name, normalized base URL, and timestamps into desktop_server_profiles.

After MarkAuthenticated(profileID), assert every legacy secret column is empty
and enterprise_resource_cache rows for that server are deleted. Assert personal
tables are byte-for-byte unchanged.

- [ ] **Step 2: Run and verify failure**

~~~powershell
go test ./internal/desktopremote -run Migration -count=1
~~~

- [ ] **Step 3: Implement idempotent two-stage migration**

MigrateMetadata runs at startup and may run repeatedly. MarkAuthenticated is
the only operation allowed to clear secrets, and it runs in one GORM
transaction after new server-native login succeeds.

- [ ] **Step 4: Run migration and store tests**

~~~powershell
go test ./internal/desktopremote ./internal/enterprise -run "Migration|Store" -count=1
~~~

- [ ] **Step 5: Commit**

~~~powershell
git add -- internal/desktopremote/migration.go internal/desktopremote/migration_test.go internal/enterprise/store.go internal/container/container.go
git commit -m "feat: migrate legacy enterprise server profiles safely"
~~~

### Task 14: One-Time Desktop Handoff Domain

**Files:**

- Create: internal/types/desktop_handoff.go
- Create: internal/types/interfaces/desktop_handoff.go
- Create: internal/application/repository/desktop_handoff.go
- Create: internal/application/repository/desktop_handoff_test.go
- Create: internal/application/service/desktop_handoff.go
- Create: internal/application/service/desktop_handoff_test.go
- Create: migrations/versioned/000064_desktop_handoff_codes.up.sql
- Create: migrations/versioned/000064_desktop_handoff_codes.down.sql
- Modify: internal/container/container.go

- [ ] **Step 1: Write failing repository consume tests**

Seed one valid, one expired, and one already-consumed row. Assert ConsumeByCode
returns the valid row once, then ErrHandoffInvalid; expired and consumed rows
always return ErrHandoffInvalid. Assert the database stores SHA-256(code), not
the plaintext code.

- [ ] **Step 2: Run and verify failure**

~~~powershell
go test ./internal/application/repository -run DesktopHandoff -count=1
~~~

- [ ] **Step 3: Add the model, migration, and atomic repository**

DesktopHandoff fields are ID, CodeHash, Kind, ReferenceID, PKCEChallenge,
ExpiresAt, ConsumedAt, CreatedAt. Consume with one transaction and a conditional
UPDATE whose WHERE clause includes code_hash, consumed_at IS NULL, and
expires_at > now. Require RowsAffected == 1 before reading the consumed row.

- [ ] **Step 4: Write and implement service tests**

The service issues 32 random bytes, persists only their hash, sets a 60-second
expiry, and returns the plaintext code once. It accepts only kind=invite or
kind=oidc and rejects an empty reference ID.

~~~powershell
go test ./internal/application/repository ./internal/application/service -run DesktopHandoff -count=1
~~~

- [ ] **Step 5: Commit**

~~~powershell
git add -- internal/types/desktop_handoff.go internal/types/interfaces/desktop_handoff.go internal/application/repository/desktop_handoff.go internal/application/repository/desktop_handoff_test.go internal/application/service/desktop_handoff.go internal/application/service/desktop_handoff_test.go migrations/versioned/000064_desktop_handoff_codes.up.sql migrations/versioned/000064_desktop_handoff_codes.down.sql internal/container/container.go
git commit -m "feat: add single-use desktop handoff codes"
~~~

### Task 15: Server OIDC PKCE And Invite Handoff Endpoints

**Files:**

- Create: internal/handler/desktop_handoff.go
- Create: internal/handler/desktop_handoff_test.go
- Modify: internal/handler/auth.go
- Modify: internal/application/service/user.go
- Modify: internal/router/router.go

- [ ] **Step 1: Write failing endpoint tests**

Assert:

- desktop OIDC start stores the PKCE challenge against OAuth state;
- callback with the wrong state is rejected;
- callback issues xelora://auth with only server origin and handoff code;
- exchange requires a verifier whose SHA-256 challenge matches;
- exchange consumes the code before generating LoginResponse;
- invite handoff references a validated invitation/share record and returns no
  original invitation token.

- [ ] **Step 2: Run and verify failure**

~~~powershell
go test ./internal/handler ./internal/application/service -run "OIDCDesktop|DesktopHandoff" -count=1
~~~

- [ ] **Step 3: Add public start and exchange routes**

The server remains the OIDC client. Add:

~~~text
POST /api/v1/auth/desktop/oidc/start
POST /api/v1/auth/desktop/handoff/exchange
POST /api/v1/auth/desktop/invitation/handoff
~~~

OIDC start accepts redirect_scheme=xelora and pkce_challenge. After the existing
IdP callback authenticates the user, issue a 60-second kind=oidc handoff
referencing user and active tenant, then redirect to xelora://auth. Exchange
verifies SHA-256(verifier), atomically consumes the code, and only then calls
the existing token generator.

- [ ] **Step 4: Run auth and invitation regression tests**

~~~powershell
go test ./internal/handler ./internal/application/service -run "OIDC|DesktopHandoff|Invitation" -count=1
~~~

- [ ] **Step 5: Commit**

~~~powershell
git add -- internal/handler/desktop_handoff.go internal/handler/desktop_handoff_test.go internal/handler/auth.go internal/application/service/user.go internal/router/router.go
git commit -m "feat: add desktop OIDC and invitation handoffs"
~~~

### Task 16: Windows Protocol And Desktop OIDC Client

**Files:**

- Create: internal/desktopremote/oidc.go
- Create: internal/desktopremote/oidc_test.go
- Modify: cmd/desktop/app.go
- Modify: cmd/desktop/main.go
- Modify: scripts/installer-personal.nsi
- Modify: frontend/src/views/auth/Login.vue
- Modify: frontend/src/router/index.ts

- [ ] **Step 1: Write failing desktop PKCE and URI parser tests**

Assert verifier/challenge randomness, single-use state, rejection of non-xelora
schemes, rejection of unknown actions, exact parsing of auth/open, and no
access, refresh, provider, invitation, or share token parameter.

- [ ] **Step 2: Run and verify failure**

~~~powershell
go test ./internal/desktopremote ./cmd/desktop -run "OIDC|DeepLink" -count=1
~~~

- [ ] **Step 3: Implement system-browser OIDC**

The manager creates verifier/challenge, calls desktop OIDC start, and opens the
returned server authorization URL. On xelora://auth, it verifies in-memory
state, sends code plus verifier to exchange, stores refresh token in Credential
Manager, and returns the token-free IdentitySnapshot.

- [ ] **Step 4: Register and dispatch xelora://**

NSIS writes HKCU\Software\Classes\xelora and
HKCU\Software\Classes\xelora\shell\open\command. The quoted command is the
installed executable followed by quoted %1. Uninstall removes only installer
owned keys. The running app receives a URI event; a new process forwards its
URI to the existing instance. Unknown server origins require confirmation.

- [ ] **Step 5: Verify frontend and commit**

~~~powershell
go test ./internal/desktopremote ./cmd/desktop -run "OIDC|DeepLink" -count=1
Set-Location frontend
npm run type-check
Set-Location ..
git add -- internal/desktopremote/oidc.go internal/desktopremote/oidc_test.go cmd/desktop/app.go cmd/desktop/main.go scripts/installer-personal.nsi frontend/src/views/auth/Login.vue frontend/src/router/index.ts
git commit -m "feat: support desktop OIDC and xelora links"
~~~

### Task 17: Remove Legacy Enterprise Active Path

**Files:**

- Modify: internal/router/router.go
- Modify: internal/container/container.go
- Modify: internal/application/service/user.go
- Modify: internal/handler/tenant_member.go
- Modify: internal/enterprise/types.go
- Modify: internal/enterprise/store.go
- Create: frontend/src/utils/enterpriseLegacyAbsence.test.mjs
- Delete: frontend/src/stores/enterprise.ts
- Delete: frontend/src/api/enterprise/index.ts
- Delete: internal/handler/enterprise.go
- Delete: internal/enterprise/connector.go
- Delete: internal/enterprise/proxy.go

- [ ] **Step 1: Add an active-route absence test**

Inspect Gin routes in personal edition and assert no path begins
/api/v1/enterprise and no route ends /client-users. Add a frontend source test
that rejects enterprise:<server> and /api/v1/enterprise outside migration
fixtures.

- [ ] **Step 2: Run and verify legacy paths fail the test**

~~~powershell
go test ./internal/router -run LegacyEnterpriseRoutesAbsent -count=1
Set-Location frontend
npm test -- src/utils/enterpriseLegacyAbsence.test.mjs
Set-Location ..
~~~

- [ ] **Step 3: Remove registrations and linked-user code**

Remove RegisterEnterpriseRoutes, NewEnterpriseHandler, connector startup,
linked-user provisioning, resource cache writes, and the client-users endpoint
added solely for automatic desktop provisioning. Keep only a read-only legacy
ServerConfig projection and Store methods required by Task 13 migration for one
release.

- [ ] **Step 4: Remove frontend legacy modules and run focused tests**

~~~powershell
go test ./internal/router ./internal/enterprise ./internal/application/service -count=1
Set-Location frontend
npm test -- src/utils/enterpriseLegacyAbsence.test.mjs src/utils/enterpriseParity.test.ts
npm run type-check
Set-Location ..
~~~

- [ ] **Step 5: Commit**

~~~powershell
git add -- internal/router/router.go internal/container/container.go internal/application/service/user.go internal/handler/tenant_member.go internal/enterprise/types.go internal/enterprise/store.go frontend/src/utils/enterpriseLegacyAbsence.test.mjs
git add -u -- frontend/src/stores/enterprise.ts frontend/src/api/enterprise/index.ts internal/handler/enterprise.go internal/enterprise/connector.go internal/enterprise/proxy.go
git commit -m "refactor: remove legacy enterprise proxy runtime"
~~~

### Task 18: Full Build And Two-Client Acceptance

**Files:**

- Create: docs/customizations/UNIFIED_ENTERPRISE_RUNTIME_VERIFICATION.md
- Modify only when failures prove a defect: files owned by Tasks 1-17

- [ ] **Step 1: Run complete automated verification**

~~~powershell
go test ./... -count=1
Set-Location frontend
npm test
npm run type-check
npm run build
Set-Location ..
powershell -ExecutionPolicy Bypass -File .\scripts\build-personal.ps1 -OutputDir .\dist\personal-unified-runtime
~~~

Expected: all tests pass, frontend production build succeeds, and Xelora
Personal.exe is produced.

- [ ] **Step 2: Verify forbidden persistence and route patterns**

~~~powershell
rg -n "enterprise:[^ ]+|/api/v1/enterprise|client-users" frontend/src internal/router internal/handler
rg -n "xelora_token|xelora_refresh_token" frontend/src/stores frontend/src/views/auth frontend/src/router
~~~

Expected: no active enterprise route/ID/provisioning matches; token-storage
matches are personal-context guarded and documented by tests.

- [ ] **Step 3: Run the real two-client matrix**

Against one real server, use two users, two tenants, Viewer, Contributor, Admin,
Owner, and one organization/shared space. Verify web-to-desktop and
desktop-to-web resource identity, one shared chat history, sharing/revocation,
context isolation, disconnect recovery, 401, 403, deletion, certificate
change, and incompatible API major behavior.

- [ ] **Step 4: Record evidence**

Write server/client versions, test identities without secrets, request IDs,
result table, screenshot paths, and any environmental exclusions to
UNIFIED_ENTERPRISE_RUNTIME_VERIFICATION.md. Do not include credentials,
refresh tokens, handoff codes, or invitation codes.

- [ ] **Step 5: Commit verification evidence**

~~~powershell
git add -- docs/customizations/UNIFIED_ENTERPRISE_RUNTIME_VERIFICATION.md
git commit -m "test: verify unified enterprise desktop runtime"
~~~

## Final Completion Gate

- [ ] Run `git diff --check`.
- [ ] Confirm each task commit contains only its explicit files.
- [ ] Confirm the approved design acceptance criteria map to a passing test or
  recorded real-environment check.
- [ ] Confirm unrelated pre-existing worktree changes remain untouched.
- [ ] Do not declare completion if the real two-client acceptance matrix has
  not run successfully; report it as a remaining external verification step.
