# Admin User Permissions Module Design

**Date:** 2026-07-28  
**Status:** Approved for implementation planning  
**Scope:** Dedicated system-admin “User permissions” settings section to list all users and manage the three client capability flags (create KB, create agent, bind model), plus defaults / tighten-existing controls.

## Problem

Platform admins need a clear place to:

1. See who is a normal user vs client-provisioned vs system admin.
2. View and change per-user create/bind capabilities for everyone.
3. Configure defaults for new client users and optionally tighten existing ones.

Today this is partially buried inside System Settings as a client-only card, and there is no full-user list for permission management. Admins also lack a simple mental model: “who is the admin?” vs “who is a managed user?”

## Goals

1. Use existing **`is_system_admin`** as the sole platform-admin flag (no new admin role in v1).
2. Admin identity is **not** tied to a fixed username such as `admin`; promotion / seed remains the source of truth.
3. Add an independent settings section **用户权限 / User permissions** under system admin UI.
4. List **all users** with read-only identity/context fields and editable capability toggles.
5. Move defaults + tighten-existing + per-user toggles into this section; remove the duplicate full card from System Settings.
6. Reuse existing capability APIs where possible; add a paginated admin user list API.

## Non-Goals (v1)

- New role type (e.g. “permissions-only admin”).
- Changing tenant RBAC roles (Owner / Admin / Contributor / Viewer) from this page.
- Promote / revoke system admin from this page (keep existing System Settings admin roster).
- Per-tenant capability overrides for client users.
- Hiding create buttons (deny UX remains contact-admin dialog + API 403).

## Locked decisions

| Topic | Choice |
|---|---|
| Who is platform admin | Existing `is_system_admin` only |
| First / additional admins | Promote / seed (option A); no fixed `admin` account |
| Module scope | User list (read identity) + three capability flags + defaults + tighten |
| UI placement | Settings section `user-permissions` (approach 1) |
| Client users | Only system admin may edit their capabilities (unchanged) |

## Roles and authority

```text
System admin (is_system_admin=true)
  → may open User permissions
  → may list all users
  → may edit any user’s can_* flags
  → may edit client capability defaults and run tighten-existing

Everyone else
  → no nav entry; section gated like other system-admin sections
```

Tenant Owner/Admin behavior for non-client members remains on Tenant Members (out of scope for this module’s new page, but must not regress).

## Information architecture

**Route:** `/platform/settings?section=user-permissions`  
**Guard:** `requiresSystemAdmin` / existing `SYSTEM_ADMIN_SECTIONS` set.

**Page layout (top → bottom):**

1. **Defaults panel**
   - Three switches: create KB / create agent / bind model for *new* client users.
   - Tighten-existing control + confirm + run action.
2. **Users table**
   - Search: username / email (`q`).
   - Filter: `all` | `client` | `non_client` | `system_admin`.
   - Columns: user (name + email), badges (client / system admin), home tenant name (read-only), three capability switches.
   - Pagination.
3. Empty / error / retry states.

**Migration of existing UI:**  
Move `ClientUserCapabilitiesSection` content into this section (or replace with `UserPermissions.vue` that absorbs it). Remove the full card from the top of `SystemSettings.vue`.

## Data model

No new tables. Reuse `users` columns:

- `is_system_admin`
- `is_client_provisioned`
- `can_create_knowledge_base`
- `can_create_agent`
- `can_bind_model`
- `capabilities_overridden_at`
- `tenant_id` (home tenant; join name for display)

System settings keys (already registered):

- `auth.client_user_capability_defaults`
- `auth.tighten_existing_client_users`

## API

All under `/api/v1/system/admin/*` with `RequireSystemAdmin`.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/users` | **New.** Paginated full user list. Query: `q`, `filter` (`all`\|`client`\|`non_client`\|`system_admin`), `offset`, `limit`. Response items include identity + capability flags + `home_tenant_id` / `home_tenant_name`. |
| `GET`/`PUT` | `/client-capabilities/defaults` | Existing defaults + tighten flag |
| `POST` | `/client-capabilities/tighten-existing` | Existing tighten action |
| `PUT` | `/users/:id/capabilities` | Existing per-user capability update |
| `GET` | `/client-users` | Keep for compatibility; new UI uses `GET /users` |

**List item shape (illustrative):**

```json
{
  "id": "...",
  "username": "...",
  "email": "...",
  "is_system_admin": false,
  "is_client_provisioned": true,
  "can_create_knowledge_base": false,
  "can_create_agent": false,
  "can_bind_model": false,
  "home_tenant_id": 1,
  "home_tenant_name": "xuwh's Workspace"
}
```

**Audit:** Continue existing audit actions on capability update and tighten; list is read-only (no audit).

## Frontend implementation notes

1. `Settings.vue`: add `user-permissions` to admin-only nav; label 用户权限 / User permissions.
2. New (or upgraded) view: defaults + filterable table; wire `listAdminUsers`.
3. `SystemSettings.vue`: remove migrated client-capabilities card.
4. i18n zh-CN + en-US for nav, filters, columns, empty/error copy.
5. Toggle failures roll back UI state and toast.

## Acceptance criteria

1. Non–system-admin users do not see the section; deep-link does not expose controls.
2. System admin sees all users; filters and search behave as specified.
3. Toggling a user’s three flags persists and matches create KB / agent / bind-model runtime gates.
4. Changing defaults affects newly provisioned client users; tighten-existing only updates non-overridden client users.
5. System Settings no longer hosts the full client-capabilities table (moved to User permissions).

## Relationship to prior work

Builds on `2026-07-28-client-user-capabilities-design.md` and its implementation (middleware dual-gate, Personal repair paths, existing admin capability endpoints). This module is the **admin UX + full user list** layer on top of that capability model.

## Out of scope follow-ups (explicit)

- Dedicated `/platform/system/users` route outside Settings.
- Permissions-only admin role.
- Managing tenant membership/roles from this page.
- Promote/revoke system admins from this page.
