# Client User Capabilities Design

**Date:** 2026-07-28  
**Status:** Draft for implementation planning  
**Scope:** Distinguish Personal/client registered users from admins by adding per-user capability gates for create knowledge base, create agent, and bind/configure models.

## Problem

Today Personal client provisioning makes each registered user **Owner** of their home tenant. Tenant RBAC then allows them to create knowledge bases, create agents, and bind models by default. Product intent is:

- Default client login users **cannot** bind models or create knowledge bases / agents.
- Clicking create still shows the action, then prompts the user to contact an administrator.
- Administrators can configure these permissions per user, with system-level defaults and optional tightening of existing client users.

## Goals

1. Default-deny the three capabilities for newly provisioned client users.
2. Keep existing home-tenant + Owner layout; do not force client users into a shared tenant for v1.
3. Dual gate: existing tenant RBAC **and** per-user capability flags.
4. System admin sets global defaults and is the only actor who can change capabilities for client-provisioned users.
5. Tenant Owner/Admin may configure capabilities for **non-client** other members (not self).
6. Extensible capability list; v1 UI exposes only three flags.
7. Optional migration switch to tighten existing client users without surprising admins by default.

## Non-Goals (v1)

- Replacing tenant RBAC (`owner` / `admin` / `contributor` / `viewer`).
- Moving Personal users out of home tenants into a shared company tenant.
- Fine-grained per-tenant capability overrides for client users (client capabilities are user-global and system-admin-owned).
- Gating upload-into-KB, edit/delete owned resources, or chat/run of existing agents beyond the three create/bind actions.
- Hiding create buttons (v1 always shows them and denies on click / API).

## Decisions (locked)

| Topic | Choice |
|---|---|
| Admin model | System admin sets defaults; tenant admin may configure non-client members; **client users only editable by system admin** |
| Capability scope | Create KB, create agent, bind model; schema/UI registry extensible |
| Personal home | Keep home tenant + Owner; capabilities default off even for Owner |
| Deny UX | Keep create/bind entry points visible; click → “contact admin” dialog |
| Existing users | Configurable “tighten existing client users” switch (default off) |

## Architecture

### Capability model

Store on `users` (columns or a dedicated jsonb / side table is an implementation detail; columns preferred for simple booleans and indexing):

| Field | Meaning |
|---|---|
| `is_client_provisioned` | User was created via Personal client register / `ProvisionClientUser` |
| `can_create_knowledge_base` | May create a knowledge base when RBAC also allows |
| `can_create_agent` | May create an agent when RBAC also allows |
| `can_bind_model` | May create/update/delete model bindings when RBAC also allows |
| `capabilities_overridden_at` | Non-null when a system admin (or allowed tenant admin for non-client users) explicitly set capabilities; protects against re-applying defaults |

System settings (platform config / system settings store):

| Key | Meaning |
|---|---|
| `client_user_capability_defaults` | Default booleans for the three capabilities (all `false` initially) |
| `tighten_existing_client_users` | When enabled, apply defaults to client users who have **not** been explicitly overridden |

Future capabilities register in a single server-side list (id, label key, default) so UI and middleware share one source of truth.

### Authorization rule

For gated write operations:

```text
allow = tenant_RBAC_allows(action)
      AND user.capability_for(action) == true
```

Examples:

- `POST /knowledge-bases` → Contributor+ **and** `can_create_knowledge_base`
- `POST /agents` (and create-equivalent copy that yields a new owned agent) → Contributor+ **and** `can_create_agent`
- Model write routes (`POST/PUT/DELETE /models`, binding/credential writes that configure models) → Admin+ **and** `can_bind_model`

Owner role alone is **not** sufficient when the capability is false.

Backend enforcement is mandatory. Frontend uses `/auth/me` flags only for UX.

### Who may mutate capabilities

| Actor | Target | Allowed |
|---|---|---|
| System admin | Any user | Yes |
| Tenant Owner/Admin | Other member in tenant, `is_client_provisioned = false` | Yes |
| Tenant Owner/Admin | Self | No |
| Tenant Owner/Admin | Client-provisioned user | No (read-only; message: configured by system admin) |
| Client user | Anyone including self | No |

Self-elevation via preferences, member self-update, or forged API bodies must be rejected.

## Provisioning and migration

### New client users

Mark `is_client_provisioned = true` when the account is created through Personal client self-register or `ProvisionClientUser` (Personal edition client onboarding). Ordinary enterprise invites / admin-created tenant members stay `false` unless explicitly marked later.

On that path:

1. Keep current home tenant creation + `EnsureOwner`.
2. Set `is_client_provisioned = true`.
3. Copy `client_user_capability_defaults` into the three `can_*` fields.
4. Leave `capabilities_overridden_at` null.

Capabilities are **user-global** (not per-tenant). If a client user later joins another tenant, the same `can_*` flags still apply there; only system admin may change them.

### Existing users

- Default: do **not** change existing users when shipping the feature.
- System setting `tighten_existing_client_users`:
  - When turned **on** (with confirmation): for each `is_client_provisioned` user with `capabilities_overridden_at IS NULL`, set the three flags from current defaults (normally all false).
  - Users with non-null `capabilities_overridden_at` keep their admin-granted values.
- Backfill `is_client_provisioned` for known client-origin users where detectable (e.g. provision path / edition metadata); users that cannot be classified remain non-client and keep current RBAC-only behavior until marked.

## API surface

| Endpoint | Auth | Purpose |
|---|---|---|
| `GET /auth/me` (+ login/refresh payloads) | Authenticated | Include `is_client_provisioned` and three `can_*` flags |
| `GET/PUT /system/client-capability-defaults` | System admin | Read/update defaults + tighten-existing switch |
| `GET/PUT /system/users/:id/capabilities` | System admin | Read/update a user's capabilities (sets override timestamp on PUT) |
| `PUT /tenants/current/members/:userId/capabilities` | Tenant Owner/Admin | Update non-client **other** member; 403 for self or client-provisioned target |

Denied gated writes return **403** with stable code `capability_denied` so the SPA can show the contact-admin dialog for both pre-checks and API failures.

Audit log entries for: default changes, tighten-existing enablement, per-user capability updates (actor, target, before/after).

## UI

1. **System admin → Client user permissions**
   - Default toggles for the three capabilities.
   - “Tighten existing client users” with confirm dialog.
   - User table filtered to client-provisioned users; per-user toggles.
2. **Tenant members**
   - Non-client members: editable capability toggles for others.
   - Client members: read-only capabilities + helper text.
3. **Create / bind entry points** (KB list, Agent list, Model settings)
   - Keep buttons visible.
   - On click without capability: dialog — current account lacks permission; contact an administrator.
   - Same copy when API returns `capability_denied`.

## Error handling

- Prefer capability check after RBAC so existing role errors stay unchanged when role is insufficient.
- Never treat missing capability fields as allow; missing/null treated as deny for client-provisioned users, and as “fall back to RBAC-only” only for non-client users if flags were never introduced for them (implementation should migrate all users to explicit booleans with safe defaults: non-client users default **true** for the three flags so behavior stays unchanged; client users default **false**).

### Default matrix for migrated rows

| User kind | `can_*` default when column added |
|---|---|
| `is_client_provisioned = true` | `false` |
| otherwise | `true` (preserve today’s RBAC-only power) |

New client provision always writes from system defaults (false/false/false unless admin changed defaults).

## Testing

1. New client user (home Owner): create KB / create agent / bind model → UI dialog + API 403 `capability_denied`.
2. System admin enables all three for that user → same operations succeed (RBAC still applies).
3. Client user cannot update own or others’ capabilities; tenant Admin updating a client member → 403.
4. Tenant Admin can update another non-client member’s capabilities; cannot update self.
5. Tighten-existing off: legacy client users (if marked) unchanged until switch on; on: non-overridden client users tightened; overridden users preserved.
6. Direct API calls without UI still 403 when capability false.
7. Non-client Contributor without capability false-path: with default `true`, behavior matches pre-feature RBAC.

## Rollout

1. Migrate schema + backfill defaults (`client` → false, others → true).
2. Wire middleware/handlers + `/auth/me`.
3. System admin UI + tenant member read-only/edit rules.
4. Frontend deny dialogs on create/bind entry points.
5. Ship tighten-existing switch default **off**; document admin runbook.

## Open implementation notes (not product questions)

- Exact storage: boolean columns on `users` vs jsonb — prefer columns for v1 three flags.
- Whether agent “copy” counts as create: **yes** when it creates a new agent owned by the caller.
- Model “bind” includes any write that registers or rebinds a model for the tenant, not merely listing models.
