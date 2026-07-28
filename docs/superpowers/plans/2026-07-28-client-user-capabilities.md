# Client User Capabilities Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Default-deny create-KB / create-agent / bind-model for Personal client users, enforce dual-gate (RBAC + per-user capabilities), let system admins configure defaults and per-user flags, and show a “contact admin” dialog when users click create without permission.

**Architecture:** Add boolean columns on `users` plus `capabilities_overridden_at`. Introduce `RequireCapability` middleware after existing RBAC on create/bind routes. Persist defaults and tighten-existing via system settings keys, exposed through dedicated SystemAdmin APIs. Frontend reads flags from `/auth/me`, keeps create buttons visible, and intercepts clicks / `capability_denied` with one shared dialog.

**Tech Stack:** Go/Gin, Postgres+SQLite migrations, existing `system_settings` + audit log, Vue 3 + Pinia + TDesign, vitest/go test.

**Spec:** `docs/superpowers/specs/2026-07-28-client-user-capabilities-design.md`

---

## File map

| File | Responsibility |
|---|---|
| `migrations/versioned/000064_client_user_capabilities.up.sql` (+ `.down.sql`) | Postgres columns + indexes |
| `migrations/sqlite/000002_client_user_capabilities.sql` | SQLite columns (follow existing sqlite migration style) |
| `internal/types/user.go` | User / UserInfo fields + `ToUserInfo` |
| `internal/types/capability.go` | Capability IDs, registry, `UserHasCapability` |
| `internal/middleware/capability.go` | `RequireCapability` → 403 `{code,message}` |
| `internal/middleware/capability_test.go` | Middleware unit tests |
| `internal/router/rbac.go` + `router.go` | Wire capability after Contributor/Admin |
| `internal/application/service/user.go` | Provision/Register apply defaults; update capabilities |
| `internal/application/service/system_setting.go` | Register default keys |
| `internal/handler/system_capabilities.go` | Defaults, tighten, list/update client user caps |
| `internal/handler/tenant_member.go` | Tenant-scoped capability update for non-client others |
| `internal/handler/auth.go` | No shape change beyond `ToUserInfo` fields |
| `frontend/src/api/auth/index.ts` | `UserInfo` fields + `userInfoFromApi` |
| `frontend/src/stores/auth.ts` | `hasCapability(id)` |
| `frontend/src/api/system/capabilities.ts` | Admin APIs |
| `frontend/src/api/tenant/members.ts` | Member capability PUT |
| `frontend/src/components/CapabilityDeniedDialog.vue` | Shared contact-admin dialog |
| `frontend/src/views/knowledge/KnowledgeBaseList.vue` | Gate create click |
| `frontend/src/views/agent/AgentList.vue` | Gate create click |
| `frontend/src/views/settings/ModelSettings.vue` | Gate add/bind click |
| `frontend/src/views/system/ClientUserCapabilities.vue` | System admin UI |
| `frontend/src/views/settings/Settings.vue` | Mount system section |
| `frontend/src/views/settings/TenantMembers.vue` | Per-member capability toggles |
| `frontend/src/i18n/locales/zh-CN.ts` + `en-US.ts` | Copy |

---

### Task 1: Schema + User model fields

**Files:**
- Create: `migrations/versioned/000064_client_user_capabilities.up.sql`
- Create: `migrations/versioned/000064_client_user_capabilities.down.sql`
- Create: `migrations/sqlite/000002_client_user_capabilities.sql` (or next available sqlite number)
- Modify: `internal/types/user.go`
- Test: compile-only this task; behavior tests in Task 3+

- [ ] **Step 1: Add Postgres migration**

```sql
-- 000064_client_user_capabilities.up.sql
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_client_provisioned BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS can_create_knowledge_base BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS can_create_agent BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS can_bind_model BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS capabilities_overridden_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_users_is_client_provisioned
    ON users (is_client_provisioned)
    WHERE deleted_at IS NULL;

COMMENT ON COLUMN users.is_client_provisioned IS 'Personal client register / ProvisionClientUser';
COMMENT ON COLUMN users.can_create_knowledge_base IS 'Dual-gate with tenant RBAC';
COMMENT ON COLUMN users.can_create_agent IS 'Dual-gate with tenant RBAC';
COMMENT ON COLUMN users.can_bind_model IS 'Dual-gate with tenant RBAC';
COMMENT ON COLUMN users.capabilities_overridden_at IS 'Set when admin explicitly overrides capabilities';
```

```sql
-- 000064_client_user_capabilities.down.sql
DROP INDEX IF EXISTS idx_users_is_client_provisioned;
ALTER TABLE users
    DROP COLUMN IF EXISTS capabilities_overridden_at,
    DROP COLUMN IF EXISTS can_bind_model,
    DROP COLUMN IF EXISTS can_create_agent,
    DROP COLUMN IF EXISTS can_create_knowledge_base,
    DROP COLUMN IF EXISTS is_client_provisioned;
```

Note: migration default `TRUE` for `can_*` preserves existing non-client users. New client provision paths overwrite to defaults (false) in service code.

- [ ] **Step 2: Mirror SQLite migration** using the same column names/types (`BOOLEAN`, `DATETIME` for overridden_at) matching `migrations/sqlite/000001_add_user_is_system_admin.sql` style.

- [ ] **Step 3: Extend `User` and `UserInfo`**

In `internal/types/user.go` on `User`:

```go
IsClientProvisioned     bool       `json:"is_client_provisioned" gorm:"default:false;index"`
CanCreateKnowledgeBase  bool       `json:"can_create_knowledge_base" gorm:"default:true"`
CanCreateAgent          bool       `json:"can_create_agent" gorm:"default:true"`
CanBindModel            bool       `json:"can_bind_model" gorm:"default:true"`
CapabilitiesOverriddenAt *time.Time `json:"capabilities_overridden_at,omitempty"`
```

Mirror the same JSON fields on `UserInfo` and copy them in `ToUserInfo()`.

- [ ] **Step 4: Commit**

```bash
git add migrations/versioned/000064_client_user_capabilities.up.sql \
  migrations/versioned/000064_client_user_capabilities.down.sql \
  migrations/sqlite/000002_client_user_capabilities.sql \
  internal/types/user.go
git commit -m "feat: add client user capability columns"
```

---

### Task 2: Capability registry helper

**Files:**
- Create: `internal/types/capability.go`
- Create: `internal/types/capability_test.go`

- [ ] **Step 1: Write failing tests**

```go
package types

import "testing"

func TestUserHasCapability(t *testing.T) {
	u := &User{
		CanCreateKnowledgeBase: false,
		CanCreateAgent:         true,
		CanBindModel:           false,
	}
	if UserHasCapability(u, CapabilityCreateKnowledgeBase) {
		t.Fatal("expected create KB denied")
	}
	if !UserHasCapability(u, CapabilityCreateAgent) {
		t.Fatal("expected create agent allowed")
	}
	if UserHasCapability(nil, CapabilityCreateAgent) {
		t.Fatal("nil user must deny")
	}
	if UserHasCapability(u, "unknown") {
		t.Fatal("unknown capability must deny")
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/types -run TestUserHasCapability -count=1`  
Expected: FAIL (undefined)

- [ ] **Step 3: Implement registry**

```go
package types

const (
	CapabilityCreateKnowledgeBase = "can_create_knowledge_base"
	CapabilityCreateAgent         = "can_create_agent"
	CapabilityBindModel           = "can_bind_model"
)

type CapabilityDef struct {
	ID           string
	DefaultFalse bool // documentation only; defaults live in system settings for client users
}

func CapabilityRegistry() []CapabilityDef {
	return []CapabilityDef{
		{ID: CapabilityCreateKnowledgeBase},
		{ID: CapabilityCreateAgent},
		{ID: CapabilityBindModel},
	}
}

func UserHasCapability(u *User, id string) bool {
	if u == nil {
		return false
	}
	switch id {
	case CapabilityCreateKnowledgeBase:
		return u.CanCreateKnowledgeBase
	case CapabilityCreateAgent:
		return u.CanCreateAgent
	case CapabilityBindModel:
		return u.CanBindModel
	default:
		return false
	}
}
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/types -run TestUserHasCapability -count=1`

- [ ] **Step 5: Commit**

```bash
git add internal/types/capability.go internal/types/capability_test.go
git commit -m "feat: add user capability registry helper"
```

---

### Task 3: `RequireCapability` middleware

**Files:**
- Create: `internal/middleware/capability.go`
- Create: `internal/middleware/capability_test.go`
- Modify: `internal/middleware/auth.go` only if user object is not already on context — prefer loading via existing user lookup pattern used elsewhere, or pass user flags into context at auth time.

**Context note:** Auth middleware today sets `UserIDContextKey`. Capability check must load the user (or cache capability flags on context during auth). Prefer extending auth to set a small capabilities snapshot on context to avoid per-request DB hits on every gated route; if that is too large for this task, load user by ID inside `RequireCapability` via an injected `UserLookup` func (same style as ownership checks).

- [ ] **Step 1: Write failing middleware tests** (pattern from `rbac_test.go`)

```go
func TestRequireCapability_DeniesWhenFalse(t *testing.T) {
	// harness seeds context user with CanCreateKnowledgeBase=false
	// GET /protected with RequireCapability(CapabilityCreateKnowledgeBase, lookup)
	// expect 403 and body containing "capability_denied"
}

func TestRequireCapability_AllowsWhenTrue(t *testing.T) {
	// CanCreateKnowledgeBase=true → 200
}
```

Response body contract (align with workspace denial style):

```json
{"code":"capability_denied","message":"Current account lacks this permission; contact an administrator"}
```

- [ ] **Step 2: Run tests — expect FAIL**

Run: `go test ./internal/middleware -run TestRequireCapability_ -count=1`

- [ ] **Step 3: Implement middleware**

```go
func RequireCapability(id string, lookup func(ctx context.Context, userID string) (*types.User, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, _ := c.Request.Context().Value(types.UserIDContextKey).(string)
		if uid == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "unauthorized"})
			return
		}
		user, err := lookup(c.Request.Context(), uid)
		if err != nil || !types.UserHasCapability(user, id) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    "capability_denied",
				"message": "Current account lacks this permission; contact an administrator",
			})
			return
		}
		c.Next()
	}
}
```

Wire `lookup` from router using `userService.GetUserByID` (or existing repo method name in this codebase).

- [ ] **Step 4: Run tests — expect PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/middleware/capability.go internal/middleware/capability_test.go
git commit -m "feat: add RequireCapability middleware"
```

---

### Task 4: Attach capability gates to routes

**Files:**
- Modify: `internal/router/rbac.go` — add helpers `CreateKB()`, `CreateAgent()`, `BindModel()` that chain role + capability, **or** chain in `router.go` call sites
- Modify: `internal/router/router.go` — `RegisterKnowledgeBaseRoutes`, `RegisterCustomAgentRoutes`, `RegisterModelRoutes`

- [ ] **Step 1: Add router helpers** (example)

```go
func (g *RBACGuards) CreateKnowledgeBase() gin.HandlerFunc {
	return chain(g.Contributor(), g.requireCap(types.CapabilityCreateKnowledgeBase))
}
```

Where `chain` runs handlers in order (Contributor first, then capability). Implement `chain` as a small local helper that invokes each until Abort.

- [ ] **Step 2: Replace route middleware**

- `POST /knowledge-bases` → `g.CreateKnowledgeBase()` (or `g.Contributor(), requireCap(...)`)
- `POST /agents` and `POST /agents/:id/copy` → create-agent capability
- `POST/PUT/DELETE /models` and model credential write routes currently on `g.Admin()` → keep Admin **and** add bind-model capability

- [ ] **Step 3: Manual smoke / handler test**

Add or extend a handler test that calls create KB as Owner with `can_create_knowledge_base=false` → 403 `capability_denied`.

Run: `go test ./internal/handler -run Capability -count=1` (or the concrete test name you add)

- [ ] **Step 4: Commit**

```bash
git add internal/router/rbac.go internal/router/router.go internal/handler/*_test.go
git commit -m "feat: dual-gate KB/agent/model writes with capabilities"
```

---

### Task 5: Provision / Register apply client defaults

**Files:**
- Modify: `internal/application/service/user.go` (`Register`, `ProvisionClientUser`)
- Modify: `internal/application/service/system_setting.go` (register keys)
- Modify: `internal/types/interfaces/user.go` if new methods are required
- Test: `internal/application/service/user_capability_test.go` (new) or extend existing user service tests

**System setting keys:**

| Key | Type | Default |
|---|---|---|
| `auth.client_user_capability_defaults` | `string` holding JSON | `{"can_create_knowledge_base":false,"can_create_agent":false,"can_bind_model":false}` |
| `auth.tighten_existing_client_users` | `string` enum `true`/`false` | `false` |

(Registry currently has `string` / `int` / `string_list` only — store JSON as string.)

- [ ] **Step 1: Register settings in `settingRegistry`**

```go
"auth.client_user_capability_defaults": {
    Type:     "string",
    Default:  `{"can_create_knowledge_base":false,"can_create_agent":false,"can_bind_model":false}`,
    Category: "auth",
    Description: "Default capabilities for newly provisioned Personal client users (JSON object).",
},
"auth.tighten_existing_client_users": {
    Type:     "string",
    Enum:     []string{"true", "false"},
    Default:  "false",
    Category: "auth",
    Description: "When true, apply defaults to client users without capabilities_overridden_at.",
},
```

- [ ] **Step 2: Helper to load defaults + apply to user**

```go
func (s *userService) applyClientCapabilityDefaults(ctx context.Context, user *types.User) error {
	user.IsClientProvisioned = true
	defs := s.loadClientCapabilityDefaults(ctx) // parse JSON from system settings
	user.CanCreateKnowledgeBase = defs.CanCreateKnowledgeBase
	user.CanCreateAgent = defs.CanCreateAgent
	user.CanBindModel = defs.CanBindModel
	user.CapabilitiesOverriddenAt = nil
	return nil
}
```

- [ ] **Step 3: Call from `ProvisionClientUser` on create path** (and on first create only; password reset path must not wipe overridden caps).

- [ ] **Step 4: Call from `Register` when `handler.Edition == "personal"`**  
  Pass edition in via constructor/config already available to the service, or set flags in the auth handler after Register returns — prefer service receiving `clientProvisioned bool` parameter or reading edition from config injected at startup.

  Enterprise `Register` (Edition standard): leave `is_client_provisioned=false` and DB defaults `can_*=true`.

- [ ] **Step 5: Tests**

- Provision new user → `IsClientProvisioned` true, three caps false (with default settings)
- Register with personal edition → same
- Register with standard edition → `IsClientProvisioned` false, caps true

- [ ] **Step 6: Commit**

```bash
git add internal/application/service/user.go \
  internal/application/service/system_setting.go \
  internal/application/service/user_capability_test.go
git commit -m "feat: apply client capability defaults on provision/register"
```

---

### Task 6: SystemAdmin capability APIs

**Files:**
- Create: `internal/handler/system_capabilities.go`
- Create: `internal/handler/system_capabilities_test.go`
- Modify: `internal/router/router.go` inside `RegisterSystemAdminRoutes`
- Modify: `internal/application/service/user.go` — `UpdateUserCapabilities`, `ListClientProvisionedUsers`, `TightenExistingClientUsers`

- [ ] **Step 1: Define request/response types**

```go
type ClientCapabilityDefaults struct {
	CanCreateKnowledgeBase bool `json:"can_create_knowledge_base"`
	CanCreateAgent         bool `json:"can_create_agent"`
	CanBindModel           bool `json:"can_bind_model"`
}

type ClientCapabilityDefaultsResponse struct {
	Defaults               ClientCapabilityDefaults `json:"defaults"`
	TightenExistingClientUsers bool                 `json:"tighten_existing_client_users"`
}

type UpdateUserCapabilitiesRequest struct {
	CanCreateKnowledgeBase bool `json:"can_create_knowledge_base"`
	CanCreateAgent         bool `json:"can_create_agent"`
	CanBindModel           bool `json:"can_bind_model"`
}
```

- [ ] **Step 2: Routes (SystemAdmin)**

```
GET  /system/admin/client-capabilities/defaults
PUT  /system/admin/client-capabilities/defaults
POST /system/admin/client-capabilities/tighten-existing   // body optional confirm; runs tighten
GET  /system/admin/client-users                           // list is_client_provisioned users
GET  /system/admin/users/:id/capabilities
PUT  /system/admin/users/:id/capabilities                 // sets capabilities_overridden_at=now
```

PUT defaults writes both JSON defaults and tighten flag (tighten flag alone does not mutate users; only `tighten-existing` endpoint does).

- [ ] **Step 3: Tighten logic**

```go
// UPDATE users SET can_*=defaults
// WHERE is_client_provisioned AND capabilities_overridden_at IS NULL AND deleted_at IS NULL
```

Write audit log entry (reuse existing system audit helper used by system settings).

- [ ] **Step 4: Tests**

1. Non-admin → 403  
2. Admin PUT user capabilities → overridden_at set; subsequent tighten does not change that user  
3. Tighten only flips non-overridden client users  

- [ ] **Step 5: Commit**

```bash
git add internal/handler/system_capabilities.go \
  internal/handler/system_capabilities_test.go \
  internal/router/router.go \
  internal/application/service/user.go
git commit -m "feat: system admin client capability APIs"
```

---

### Task 7: Tenant member capability API

**Files:**
- Modify: `internal/handler/tenant_member.go`
- Modify: `internal/router/router.go` (tenant member routes, Owner/Admin)
- Test: `internal/handler/tenant_member_capability_test.go`

- [ ] **Step 1: Add route**

`PUT /tenants/:id/members/:userId/capabilities` with `g.Admin()` (Admin+ includes Owner).

- [ ] **Step 2: Handler rules**

```go
// reject if actor.ID == targetUserID
// reject if target.IsClientProvisioned
// else update caps + set capabilities_overridden_at
```

- [ ] **Step 3: Tests for self / client target / success**

- [ ] **Step 4: Commit**

```bash
git add internal/handler/tenant_member.go \
  internal/handler/tenant_member_capability_test.go \
  internal/router/router.go
git commit -m "feat: tenant admin capability updates for non-client members"
```

---

### Task 8: Frontend auth types + `hasCapability`

**Files:**
- Modify: `frontend/src/api/auth/index.ts` (`UserInfo`, `userInfoFromApi`)
- Modify: `frontend/src/stores/auth.ts`
- Test: existing auth store tests if present; else add `frontend/src/stores/auth.capability.test.ts`

- [ ] **Step 1: Extend `UserInfo`**

```ts
is_client_provisioned?: boolean
can_create_knowledge_base?: boolean
can_create_agent?: boolean
can_bind_model?: boolean
```

Map all four in `userInfoFromApi` (default non-client missing flags to `true` for forward-compat with old servers; if `is_client_provisioned === true` and a flag is missing, default `false`).

- [ ] **Step 2: Store helper**

```ts
function hasCapability(id: 'can_create_knowledge_base' | 'can_create_agent' | 'can_bind_model'): boolean {
  const u = user.value
  if (!u) return false
  if (u.is_client_provisioned) {
    return u[id] === true
  }
  // non-client: missing → allow (RBAC still applies in UI via hasRole)
  return u[id] !== false
}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/auth/index.ts frontend/src/stores/auth.ts
git commit -m "feat: expose user capabilities in auth store"
```

---

### Task 9: Capability denied dialog + create entry points

**Files:**
- Create: `frontend/src/components/CapabilityDeniedDialog.vue`
- Modify: `KnowledgeBaseList.vue` (`handleCreateKnowledgeBase`)
- Modify: `AgentList.vue` (`handleCreateAgent`)
- Modify: `ModelSettings.vue` (`openAddDialog` and any other bind/create entry)
- Modify: `frontend/src/i18n/locales/zh-CN.ts`, `en-US.ts`
- Modify: global API error interceptor only if easy — optional; at minimum handle in the three click paths + show dialog on create API 403 with `code === 'capability_denied'`

- [ ] **Step 1: Dialog component**

Props: `visible`. Content keys:

- zh: `当前账号无此权限，请联系管理员`
- en: `Your account does not have this permission. Please contact an administrator.`

Use `t-dialog` with a single OK button.

- [ ] **Step 2: Gate clicks**

```ts
function handleCreateKnowledgeBase() {
  if (!authStore.hasCapability('can_create_knowledge_base')) {
    capabilityDeniedVisible.value = true
    return
  }
  uiStore.openCreateKB()
}
```

Same pattern for agent + model. **Do not hide buttons.**

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/CapabilityDeniedDialog.vue \
  frontend/src/views/knowledge/KnowledgeBaseList.vue \
  frontend/src/views/agent/AgentList.vue \
  frontend/src/views/settings/ModelSettings.vue \
  frontend/src/i18n/locales/zh-CN.ts \
  frontend/src/i18n/locales/en-US.ts
git commit -m "feat: show contact-admin dialog when capability missing"
```

---

### Task 10: System admin UI for defaults + client users

**Files:**
- Create: `frontend/src/api/system/capabilities.ts`
- Create: `frontend/src/views/system/ClientUserCapabilities.vue`
- Modify: `frontend/src/views/settings/Settings.vue` — render under `system-global` or as a subsection inside `SystemSettings.vue`
- Modify: i18n

**Preferred UX:** Add a card/section at the top of `SystemSettings.vue` (already SystemAdmin-only) titled “客户端用户权限”, rather than a new settings nav key — fewer router changes.

- [ ] **Step 1: API client**

```ts
export function getClientCapabilityDefaults()
export function updateClientCapabilityDefaults(body)
export function tightenExistingClientUsers()
export function listClientUsers()
export function updateUserCapabilities(userId, body)
```

- [ ] **Step 2: UI controls**

1. Three switches bound to defaults (persist on change with confirm if needed)
2. “收紧存量客户端用户” button → `t-popconfirm` → `tightenExistingClientUsers()`
3. Table of client users with three switches each → `updateUserCapabilities`

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/system/capabilities.ts \
  frontend/src/views/system/ClientUserCapabilities.vue \
  frontend/src/views/system/SystemSettings.vue \
  frontend/src/i18n/locales/zh-CN.ts \
  frontend/src/i18n/locales/en-US.ts
git commit -m "feat: system admin UI for client user capabilities"
```

---

### Task 11: Tenant members capability toggles

**Files:**
- Modify: `frontend/src/api/tenant/members.ts`
- Modify: `frontend/src/views/settings/TenantMembers.vue`
- Modify: i18n
- Ensure member list API returns capability fields for each user (extend tenant member list DTO if missing)

- [ ] **Step 1: Backend list payload** — include the four fields on each member user object returned by list-members (modify serializer if it strips unknown user fields).

- [ ] **Step 2: UI**

For each member row (not self):

- If `is_client_provisioned`: show three read-only switches + tip “由系统管理员配置”
- Else: editable switches calling `PUT .../capabilities`

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/tenant/members.ts \
  frontend/src/views/settings/TenantMembers.vue \
  internal/handler/tenant_member.go \
  frontend/src/i18n/locales/zh-CN.ts \
  frontend/src/i18n/locales/en-US.ts
git commit -m "feat: tenant member capability toggles for non-client users"
```

---

### Task 12: End-to-end verification checklist

**Files:** none required beyond fixes found during verification

- [ ] **Step 1: Migrate local DB and restart API**

- [ ] **Step 2: Manual checklist (from spec Testing)**

1. New Personal client user (Owner): create KB / agent / bind model → dialog + API 403 `capability_denied`
2. System admin enables three flags → operations succeed
3. Client user cannot change capabilities; tenant Admin → client member update 403
4. Tenant Admin updates another non-client member; cannot update self
5. Tighten off: no mass change; tighten on: non-overridden client users tightened; overridden preserved
6. Curl create KB with capability false → 403 even if UI bypassed
7. Non-client Contributor with default `can_*=true` behaves as before

- [ ] **Step 3: Commit any fixes**

```bash
git commit -m "fix: client capability edge cases from verification"
```

---

## Spec coverage self-review

| Spec requirement | Task |
|---|---|
| User columns + overridden_at | Task 1 |
| Capability registry extensible | Task 2 |
| Dual-gate middleware + 403 code | Task 3–4 |
| Provision/Register defaults | Task 5 |
| System defaults + tighten switch | Task 5–6 |
| System admin per-user config + list | Task 6, 10 |
| Tenant admin non-client others only | Task 7, 11 |
| `/auth/me` flags | Task 1 + 8 (via `ToUserInfo`) |
| Keep buttons + contact admin dialog | Task 9 |
| Non-client migrate default true / client false | Task 1 + 5 |
| Audit on admin changes | Task 6 |
| Agent copy gated | Task 4 |
| Model write gated | Task 4 |

No TBD placeholders remain. Type/field names are consistent: `can_create_knowledge_base`, `can_create_agent`, `can_bind_model`, `is_client_provisioned`, `capability_denied`.
