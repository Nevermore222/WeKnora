# Admin User Permissions Module Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a system-admin-only Settings section「用户权限」that lists all users and manages the three capability flags, defaults, and tighten-existing — moving the client-capabilities card out of System Settings.

**Architecture:** Extend `/api/v1/system/admin` with a filtered paginated `GET /users` that joins home tenant names. Frontend adds `user-permissions` to `SYSTEM_ADMIN_SECTIONS`, renders a new `UserPermissions.vue` (defaults + full user table), and removes `ClientUserCapabilitiesSection` from `SystemSettings.vue`. Reuse existing defaults / tighten / `PUT .../capabilities` endpoints.

**Tech Stack:** Go (Gin, GORM), Vue 3 + Pinia + TDesign, existing `frontend/src/api/system/capabilities.ts`.

**Spec:** `docs/superpowers/specs/2026-07-28-admin-user-permissions-design.md`

---

## File map

| File | Responsibility |
|---|---|
| `internal/types/interfaces/user.go` | Add `ListAdminUsers` on service + repo |
| `internal/application/repository/user.go` | Implement filtered list + count + tenant name join |
| `internal/application/service/user.go` | Thin service wrapper |
| `internal/handler/client_user_capabilities.go` (or new `admin_users.go`) | `ListAdminUsers` HTTP handler |
| `internal/handler/admin_users_test.go` | Handler tests for filter / search |
| `internal/router/router.go` | Register `GET /system/admin/users` **before** `/users/:id/...` |
| `frontend/src/api/system/capabilities.ts` | Add `listAdminUsers` + row type with tenant fields |
| `frontend/src/views/system/UserPermissions.vue` | New section UI |
| `frontend/src/views/settings/Settings.vue` | Nav + section mount + `SYSTEM_ADMIN_SECTIONS` |
| `frontend/src/views/system/SystemSettings.vue` | Remove migrated card |
| `frontend/src/views/system/ClientUserCapabilitiesSection.vue` | Delete after migration (or keep unused briefly — prefer delete) |
| `frontend/src/i18n/locales/zh-CN.ts`, `en-US.ts` | Nav + page strings |

---

### Task 1: Repository — filtered admin user list

**Files:**
- Modify: `internal/types/interfaces/user.go`
- Modify: `internal/application/repository/user.go`
- Test: `internal/application/repository/user_admin_list_test.go` (or extend existing user repo tests if present)

- [ ] **Step 1: Add interface methods**

In `UserRepository` and `UserService`:

```go
// AdminUserListFilter selects which users to include.
// Empty or "all" = no flag filter.
type AdminUserListFilter string

const (
	AdminUserFilterAll          AdminUserListFilter = "all"
	AdminUserFilterClient       AdminUserListFilter = "client"
	AdminUserFilterNonClient    AdminUserListFilter = "non_client"
	AdminUserFilterSystemAdmin  AdminUserListFilter = "system_admin"
)

// AdminUserListItem is a list row with home tenant display name.
type AdminUserListItem struct {
	User           *types.User
	HomeTenantName string
}

ListAdminUsers(ctx context.Context, q string, filter AdminUserListFilter, offset, limit int) ([]AdminUserListItem, int64, error)
```

Place the filter type + item struct in `internal/types/` (e.g. `admin_user.go`) or next to interfaces — prefer `internal/types/admin_user.go` so handler/JSON DTOs stay clean.

- [ ] **Step 2: Implement repository query**

```go
func (r *userRepository) ListAdminUsers(
	ctx context.Context, q string, filter types.AdminUserListFilter, offset, limit int,
) ([]types.AdminUserListItem, int64, error) {
	base := r.db.WithContext(ctx).Model(&types.User{}).Where("deleted_at IS NULL")
	q = strings.TrimSpace(q)
	if q != "" {
		like := "%" + q + "%"
		base = base.Where("username ILIKE ? OR email ILIKE ?", like, like)
		// SQLite: use LIKE with lower() if dialector is sqlite — mirror SearchUsers pattern in this repo.
	}
	switch filter {
	case types.AdminUserFilterClient:
		base = base.Where("is_client_provisioned = ?", true)
	case types.AdminUserFilterNonClient:
		base = base.Where("is_client_provisioned = ?", false)
	case types.AdminUserFilterSystemAdmin:
		base = base.Where("is_system_admin = ?", true)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []*types.User
	query := base.Order("created_at DESC, id ASC")
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	if err := query.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	// Batch-load tenant names by TenantID; missing tenants → empty string.
	// Return []AdminUserListItem{User, HomeTenantName}.
}
```

**SQLite note:** Personal edition uses SQLite — do **not** use `ILIKE`. Follow existing `SearchUsers` in the same repository for portable case-insensitive match.

- [ ] **Step 3: Service wrapper**

```go
func (s *userService) ListAdminUsers(...) ([]types.AdminUserListItem, int64, error) {
	return s.userRepo.ListAdminUsers(ctx, q, filter, offset, limit)
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/types/admin_user.go internal/types/interfaces/user.go \
  internal/application/repository/user.go internal/application/service/user.go
git commit -m "feat: list admin users with capability filters"
```

---

### Task 2: HTTP handler + route

**Files:**
- Modify: `internal/handler/client_user_capabilities.go` (add `ListAdminUsers`) **or** Create: `internal/handler/admin_users.go`
- Create: `internal/handler/admin_users_test.go`
- Modify: `internal/router/router.go`

- [ ] **Step 1: Handler**

```go
func (h *SystemHandler) ListAdminUsers(c *gin.Context) {
	ctx := c.Request.Context()
	q := c.Query("q")
	filter := types.AdminUserListFilter(c.DefaultQuery("filter", "all"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, total, err := h.userSvc.ListAdminUsers(ctx, q, filter, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rows := make([]gin.H, 0, len(items))
	for _, it := range items {
		u := it.User
		rows = append(rows, gin.H{
			"id":                        u.ID,
			"username":                  u.Username,
			"email":                     u.Email,
			"is_system_admin":           u.IsSystemAdmin,
			"is_client_provisioned":     u.IsClientProvisioned,
			"can_create_knowledge_base": u.CanCreateKnowledgeBase,
			"can_create_agent":          u.CanCreateAgent,
			"can_bind_model":            u.CanBindModel,
			"home_tenant_id":            u.TenantID,
			"home_tenant_name":          it.HomeTenantName,
		})
	}
	c.JSON(http.StatusOK, gin.H{"users": rows, "total": total})
}
```

- [ ] **Step 2: Register route BEFORE parameterized paths**

In `RegisterSystemAdminRoutes`:

```go
adminRoutes.GET("/users", handler.ListAdminUsers)
adminRoutes.GET("/users/:id/capabilities", handler.GetUserCapabilities)
adminRoutes.PUT("/users/:id/capabilities", handler.UpdateUserCapabilities)
```

If `/users/:id/capabilities` is registered first, `GET /users` may be shadowed depending on Gin version — put static `/users` first.

- [ ] **Step 3: Tests**

```go
func TestListAdminUsers_FilterClient(t *testing.T) {
	// stub userSvc.ListAdminUsers returns one client user
	// GET /system/admin/users?filter=client → 200, total=1, is_client_provisioned true
}

func TestListAdminUsers_SearchQ(t *testing.T) {
	// assert q passed through to service
}
```

Use the same SystemHandler test harness pattern as other system admin tests if present; otherwise gin test router with stub `UserService`.

- [ ] **Step 4: Commit**

```bash
git add internal/handler/admin_users.go internal/handler/admin_users_test.go internal/router/router.go
git commit -m "feat: GET /system/admin/users for permission management"
```

---

### Task 3: Frontend API client

**Files:**
- Modify: `frontend/src/api/system/capabilities.ts`

- [ ] **Step 1: Extend types and add listAdminUsers**

```ts
export type AdminUserFilter = 'all' | 'client' | 'non_client' | 'system_admin'

export interface AdminUserRow {
  id: string
  username: string
  email: string
  is_system_admin?: boolean
  is_client_provisioned?: boolean
  can_create_knowledge_base?: boolean
  can_create_agent?: boolean
  can_bind_model?: boolean
  home_tenant_id?: number
  home_tenant_name?: string
}

export async function listAdminUsers(params?: {
  q?: string
  filter?: AdminUserFilter
  offset?: number
  limit?: number
}): Promise<{ users: AdminUserRow[]; total: number }> {
  const qs = new URLSearchParams()
  if (params?.q) qs.set('q', params.q)
  if (params?.filter) qs.set('filter', params.filter)
  if (params?.offset != null) qs.set('offset', String(params.offset))
  if (params?.limit != null) qs.set('limit', String(params.limit))
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  return (await get(`/api/v1/system/admin/users${suffix}`)) as unknown as {
    users: AdminUserRow[]
    total: number
  }
}
```

Keep existing `listClientUsers` for compatibility (unused by new UI).

- [ ] **Step 2: Commit**

```bash
git add frontend/src/api/system/capabilities.ts
git commit -m "feat: frontend listAdminUsers API client"
```

---

### Task 4: UserPermissions.vue page

**Files:**
- Create: `frontend/src/views/system/UserPermissions.vue`
- Modify: `frontend/src/i18n/locales/zh-CN.ts`, `en-US.ts`

- [ ] **Step 1: Build page from ClientUserCapabilitiesSection**

Structure:

1. Section header: `t('settings.userPermissions.title')` + description  
2. Defaults block (reuse switch + tighten UX from `ClientUserCapabilitiesSection.vue`)  
3. Toolbar: `t-input` search (debounce 300ms), `t-select` filter  
4. `t-table` with columns: user, badges, home tenant, three switches  
5. Pagination (`offset`/`limit` or page index)

Key handlers:

```ts
async function loadUsers() {
  const resp = await listAdminUsers({
    q: searchQ.value || undefined,
    filter: filter.value,
    offset: (page.value - 1) * pageSize.value,
    limit: pageSize.value,
  })
  users.value = resp.users ?? []
  total.value = resp.total ?? 0
}

async function onUserCap(row: AdminUserRow, key: keyof ClientCapabilityFlags, value: boolean) {
  const next = {
    can_create_knowledge_base: !!row.can_create_knowledge_base,
    can_create_agent: !!row.can_create_agent,
    can_bind_model: !!row.can_bind_model,
    [key]: value,
  }
  try {
    const updated = await updateUserCapabilities(row.id, next)
    Object.assign(row, updated)
  } catch (e: any) {
    MessagePlugin.error(...)
    await loadUsers()
  }
}
```

Badge cell: show `t-tag` for `is_client_provisioned` and `is_system_admin`.

- [ ] **Step 2: i18n keys**

Under `settings`:

```ts
// zh-CN
userPermissions: {
  title: '用户权限',
  description: '查看全部用户并管理创建知识库、创建智能体、绑定模型等能力。仅系统管理员可访问。',
  filterAll: '全部用户',
  filterClient: '客户端用户',
  filterNonClient: '非客户端用户',
  filterSystemAdmin: '系统管理员',
  searchPlaceholder: '按用户名或邮箱搜索',
  colUser: '用户',
  colTenant: '主空间',
  colBadges: '标识',
  badgeClient: '客户端',
  badgeSystemAdmin: '系统管理员',
  // reuse capability.* and system.globalSettings.clientCapabilities.* for defaults/tighten
}
```

Mirror in en-US.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/system/UserPermissions.vue \
  frontend/src/i18n/locales/zh-CN.ts frontend/src/i18n/locales/en-US.ts
git commit -m "feat: UserPermissions settings page UI"
```

---

### Task 5: Wire Settings nav + remove duplicate card

**Files:**
- Modify: `frontend/src/views/settings/Settings.vue`
- Modify: `frontend/src/views/system/SystemSettings.vue`
- Delete: `frontend/src/views/system/ClientUserCapabilitiesSection.vue` (after import removed)

- [ ] **Step 1: Settings.vue**

```ts
import UserPermissions from '@/views/system/UserPermissions.vue'

const SYSTEM_ADMIN_SECTIONS = new Set(['system-global', 'user-permissions'])

// in all nav items:
{ key: 'user-permissions', icon: 'usergroup', label: t('settings.userPermissions.title') },

// platform group:
items: pickItems(['system-global', 'user-permissions', 'system']),

// content-wrapper--full when user-permissions too (same as system-global)

// template:
<div v-if="currentSection === 'user-permissions'" class="section">
  <UserPermissions />
</div>
```

Also extend `content-wrapper--full` class binding if it currently only checks `system-global`.

- [ ] **Step 2: SystemSettings.vue**

Remove:

```vue
<ClientUserCapabilitiesSection />
```

and its import.

- [ ] **Step 3: Delete** `ClientUserCapabilitiesSection.vue` if unused.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/views/settings/Settings.vue \
  frontend/src/views/system/SystemSettings.vue \
  frontend/src/views/system/ClientUserCapabilitiesSection.vue
git commit -m "feat: add user-permissions settings section; remove duplicate card"
```

---

### Task 6: Verification

**Files:** none required beyond fixes

- [ ] **Step 1: Backend tests**

```bash
go test ./internal/handler/ -run "ListAdminUsers" -count=1
go test ./internal/application/repository/ -run "ListAdminUsers" -count=1
go build ./internal/...
```

Expected: PASS / build OK.

- [ ] **Step 2: Manual checklist (web or Personal against Docker admin)**

1. Non–system-admin: no「用户权限」nav item.  
2. System admin: open Settings → 用户权限; see defaults + full user list.  
3. Filter client / non_client / system_admin; search by email.  
4. Toggle a user’s `can_create_knowledge_base` off → that user gets capability denied dialog on create KB.  
5. Toggle on → create succeeds (if tenant RBAC allows).  
6. System Settings page no longer shows the old client-capabilities card.  
7. Tighten-existing still works from the new page.

- [ ] **Step 3: Commit any fixes**

```bash
git commit -m "fix: admin user permissions verification follow-ups"
```

---

## Spec coverage

| Spec item | Task |
|---|---|
| `is_system_admin` only, no new role | Task 5 guard |
| Settings section `user-permissions` | Task 5 |
| Full user list + filters + search | Task 1–4 |
| Defaults + tighten | Task 4 (reuse APIs) |
| Per-user capability toggles | Task 4 |
| Move card out of SystemSettings | Task 5 |
| `GET /users` before `:id` routes | Task 2 |
| No tenant role / promote-revoke in v1 | Out of scope |

No TBD placeholders remain.
