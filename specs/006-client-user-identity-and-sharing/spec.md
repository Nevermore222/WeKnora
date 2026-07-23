# 006 - Client User Identity & Space Sharing

Status: Draft (confirmed model direction with user)
Created: 2026-07-22
Branch: 002-session-workspace-binding
Depends on: 005-client-server-sync (Phase 1/2 provision+shared-access foundation)
Revises: 005 Phase 1 provisioning co-tenancy choice

## 1. Background / Problem

005 Phase 1 delivered auto-provisioning + shared-resource access for the
personal client (T-501/T-502 done). Its ProvisionClientUser deliberately
creates the client user directly into the admin tenant (the tenant the
API token is bound to), instead of giving the user an independent home
tenant - that choice was made to avoid an orphan home tenant polluting the
org invite search.

That co-tenancy model conflicts with the intended model:

- Cannot share arbitrary spaces with the client user. Organization invite
  is tenant-scoped (SearchTenantsForInvite groups candidates by
  User.TenantID). The client user TenantID == admin tenant, so there is
  no independent client-user tenant to invite into an org; the admin can
  only share spaces with their own tenant, not specifically with the
  client user.
- No per-space permission control over the client user. Because the user
  is a member of the admin tenant (role contributor), they access the
  admin tenant entire KB/agent set directly, bypassing the
  OrgMemberRole (viewer/editor/admin) gating that org sharing enforces.
- No own workspace for the client user. A KB/agent the user creates lands
  in the admin tenant, not in a workspace the user owns.

## 2. Goal

Make the provisioned client user a normal, independent server user with an
own home tenant, whose permissions over shared spaces are governed by
the existing organization machinery, and who can also create their own
knowledge bases and agents on the server.

1. Provisioned user identity is owned and managed by the server/admin.
2. Admin controls the client user per-space permissions via OrgMemberRole
   by inviting the user home tenant into an organization with a role.
3. Admin can share any space with the client user, because the user has a
   discoverable independent home tenant.
4. Shared-space KBs/agents are usable by the client via the linked JWT
   (already works in T-502; unchanged after the fix).
5. The client user can create their own KBs/agents on the server inside
   their home tenant, using their linked JWT (option A, confirmed).

## 3. Design

### 3.1 Server - provisioning identity model

ProvisionClientUser (internal/application/service/user.go) changes from
create-user-in-admin-tenant to create-user-with-own-home-tenant:

- Create a tenant named <username> Workspace (same shape as Register).
- Create the user with TenantID = new home tenant.
- Ensure an Owner membership for the user in their home tenant
  (memberService.EnsureOwner), so they can create/manage their own
  resources under g.Contributor() / g.OwnedKBOrAdmin() gates.
- Do NOT add the user as a member of the admin tenant. The admin can
  later invite the user home tenant into an org to share spaces - that
  replaces the old implicit co-tenancy access.
- Idempotency: if the user already exists, keep their home tenant (do not
  re-home them into the admin tenant - the old re-homing branch removed);
  only reset the password so the client can re-authenticate.

The provisioning response returns the home tenant id as tenant_id (the
client stores it as linked_tenant_id). The API token still authorizes the
provisioning call itself; the resulting linked JWT belongs to a user whose
home tenant is independent of the admin.

Reverts the orphan-home-tenant avoidance: the isolated home tenant is now
the correct design, not a bug. SearchTenantsForInvite already resolves a
name for candidates whose home tenant is the user Workspace tenant - that
remains the discovery path.

### 3.2 Client - connect flow reorder

Current order: Connect -> discover with API token (sees admin tenant own
KBs/agents - the co-tenancy leak); ProvisionUser -> JWT;
RefreshSharedResources -> shared via JWT.

New order:
1. Connect -> lightweight reachability check (no full discover yet),
   mark connected so provision can use the live connection.
2. ProvisionUser -> creates home-tenant user, obtains JWT (persisted).
3. discover using the linked JWT -> fetches the user own home-tenant
   KBs/agents/skills (the ones they created on the server; empty initially).
4. RefreshSharedResources -> shared KBs/agents via JWT, merged into
   capabilities.

Result: ServerCapabilities holds two distinct buckets - own (server-side
home-tenant resources) and shared (org-shared resources, flagged
Shared=true) - both rendered in the unified selectors with appropriate
origin badges.

discover switches from API-token auth to JWT auth: replace the
fetchRemoteList(..., APIToken) calls with a JWT-authenticated fetch, so
the server resolves the client user home tenant, not the admin.

### 3.3 Client - enterprise proxy CRUD (option A)

The client creates/manages KBs and agents on the server through the linked
JWT. New enterprise proxy routes in RegisterEnterpriseRoutes forward to
the server existing KB/agent endpoints - the server already enforces
ownership/role gates against the linked JWT user+tenant, so no new
server-side authorization logic is needed.

Knowledge bases (proxy to server /api/v1/knowledge-bases):
- POST   /enterprise/knowledge-bases
- GET    /enterprise/knowledge-bases
- GET    /enterprise/knowledge-bases/:id
- PUT    /enterprise/knowledge-bases/:id
- DELETE /enterprise/knowledge-bases/:id
- POST   /enterprise/knowledge-bases/:id/knowledge/file (multipart;
  forwarded via ForwardRequest, not ForwardJSON)
- GET    /enterprise/knowledge-bases/:id/knowledge

Agents (proxy to server /api/v1/agents):
- POST   /enterprise/agents
- GET    /enterprise/agents
- GET    /enterprise/agents/:id
- PUT    /enterprise/agents/:id
- DELETE /enterprise/agents/:id

The proxy already prefers the linked JWT (applyAuth) and auto-refreshes on
401 (ForwardJSON), so identity flows automatically. Multipart upload
reuses ForwardRequest (streams body + Content-Type through). All paths
pass X-Enterprise-Server-ID to select the connection.

### 3.4 Frontend - server-side KB/agent creation entry points

Vue store/views mark enterprise-origin KB/agent creation as create-on-
server: the create dialog posts to /enterprise/* proxy routes instead of
local API when the selected origin is an enterprise server. Existing
ResourceOriginBadge distinguishes own-server vs shared vs local. No new
resource types - the selectors already merge enterprise resources; this
adds write paths alongside the existing chat/retrieval paths.

## 4. Non-goals

- Bidirectional sync / conflict resolution (T-503 one-way push remains
  separate). Client-local KBs are not mirrored to the server (A means
  create-on-server, not local-then-push). The server organization/share
  APIs are unchanged; this only fixes whose tenant the user lands in.

## 5. Risks

- Existing provisioned users (admin-tenant) become stale: they keep old
  admin-tenant membership; on next re-provision, leave the old membership
  (harmless) - do not re-home; their JWT still resolves their stored home
  tenant.
- discover with empty home tenant shows no resources: UI clearly separates
  my server KBs vs shared with me; shared list populates via
  RefreshSharedResources.
- Admin must now explicitly invite the user home tenant: sharing is no
  longer automatic; required by design; document the invite step; the
  user home tenant is searchable by username/email.
- Multipart upload streaming through proxy: large files / timeouts; 120s
  proxy timeout already set; chunked upload is a later concern.
