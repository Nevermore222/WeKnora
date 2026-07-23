# 006 Implementation Plan

All paths relative to repo root. Phases are ordered; each phase is independently
buildable/verifiable.

## Phase 1 - Server provisioning identity fix

Files: internal/application/service/user.go (ProvisionClientUser)

- ProvisionClientUser: create home tenant (Tenant{Name: <username> Workspace})
  via tenantService.CreateTenant, create user with TenantID = home tenant id,
  EnsureOwner membership for the user in the home tenant.
- Remove the create-user-directly-in-target-tenant branch and the re-homing
  branch; keep the existing-user password reset.
- Do NOT call AddMember into the admin (target) tenant.
- Return {user_id, email, tenant_id: home tenant id} - tenant_id now reflects
  the user own home tenant.
- Keep tenant role input for API compatibility but it is no longer used to
  join the admin tenant (used only if a caller explicitly passes it; default
  remains contributor, applied within the home tenant as the Owner membership
  which supersedes it). Simplify: ignore role for admin-tenant membership;
  user is Owner of their own home tenant.

Verify: go build ./internal/... ; ProvisionClientUser unit test asserting the
user ends up with a home tenant != the admin tenant and Owner membership there.

## Phase 2 - Client connect flow reorder + JWT discovery

Files: internal/enterprise/connector.go (Connect, discover, ProvisionUser),
internal/handler/enterprise.go (ConnectServer)

- Connect: split into reachability-only (keep a /auth/me or /health probe) and
  defer discover until after provisioning. Keep connection registered + status
  connected at reachability so ProvisionUser can use GetConnection.
- ProvisionUser: unchanged externally (still discover-tenant -> provision ->
  login -> SaveLinkedIdentity), but the returned tenant_id is now the user home
  tenant.
- discover: replace fetchRemoteList(..., conn.Config.APIToken) with a JWT-aware
  fetch that uses conn.Config.ServerJWT when present (fall back to APIToken
  only when not yet linked). introduce fetchRemoteListWithAuth(client, url,
  jwt, apiToken) or reuse fetchJSONWithJWT for kb/agent/skill lists.
- ConnectServer handler reorder: Connect (reachability) -> ProvisionUser ->
  discover-with-JWT -> RefreshSharedResources. If ProvisionUser fails, log and
  still attempt discover with whatever auth is available (best-effort, as
  today).

Verify: connector_test (or new) asserting discover called after ProvisionUser
and uses Authorization: Bearer when JWT present.

## Phase 3 - Enterprise proxy CRUD routes

Files: internal/handler/enterprise.go, internal/enterprise/proxy.go,
internal/router/router.go (RegisterEnterpriseRoutes)

- Add Proxy generic helpers if needed:
  - ProxyJSON(method, path) for KB/agent CRUD (reuse ForwardJSON).
  - ProxyMultipart(path) for document upload (reuse ForwardRequest, stream
    c.Request.Body + forward Content-Type/Accept).
- Handlers:
  - KB: CreateKB(ListKBs), GetKB, UpdateKB, DeleteKB, ListKnowledge, UploadKBFile.
  - Agent: CreateAgent, ListAgents, GetAgent, UpdateAgent, DeleteAgent.
  - All read serverID from X-Enterprise-Server-ID; no body parsing on the
    client (pure passthrough) to keep the client thin.
- Routes added in RegisterEnterpriseRoutes under /enterprise.

Verify: go build; a proxy unit test stubbing an httptest server asserting KB
create returns the upstream response with Authorization forwarded.

## Phase 4 - Frontend write entry points

Files: frontend/src/api/enterprise/* (new), frontend/src/views/... (KB create
dialog / agent create dialog origin switch), frontend/src/i18n locales

- enterprise API client: kb.ts (create/list/get/update/delete/upload), agent.ts
  (CRUD).
- Origin switch in KB/agent create dialogs: when origin === enterprise server,
  POST to /enterprise/*; reuse existing form components.
- ResourceOriginBadge: own-server variant (distinct from shared).
- i18n keys for server-side creation labels.

Verify: frontend npm run build; vue-tsc passes.

## Phase 5 - Docs + task board

Files: docs/customizations/TASKS.md, docs/customizations/CHANGELOG-custom.md,
AGENTS.md required-reading pointer

- Add T-504 (identity model fix), T-505 (proxy CRUD), T-506 (frontend writes).
- CHANGELOG entry.
- AGENTS.md: add 006 spec to required reading for provisioning/identity work.

## Out of scope (handled by T-503 later)

- One-way push of client-local KBs/agents/documents to the server
  (local-then-push). This plan covers direct create-on-server (option A) only.
