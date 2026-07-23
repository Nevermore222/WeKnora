# 005 — Client–Server Sync (客户端-服务器同步)

Status: Draft (pending user approval)
Created: 2026-07-21
Branch: 002-session-workspace-binding
Depends on: 004-personal-client (enterprise connector)

## 1. Background / Problem

The personal client and the enterprise server are two **independent instances**:

- Client: local SQLite, its own users/tenants/KBs/agents.
- Server: PostgreSQL, its own users/tenants/KBs/agents.

The client connects to the server with an **API token that is bound to a single
server tenant** (not a user). The server attributes every client request to
"that tenant's owner (or a synthetic user)" — it cannot see the client's local
users at all.

The server's sharing model is **organization-based**: KBs/agents are shared to
an *organization (space)*, and organization members are **tenants**. So a server
admin cannot "share a space with a client user" — the client user does not exist
on the server.

## 2. Goal

Let a client user obtain a **real identity on the server**, so that:

1. Server admins can share spaces/KBs/agents with the client user (as a normal
   server user).
2. The client can access those shared resources.
3. The client can push its locally-created KBs/agents up to the server.

## 3. Chosen model (confirmed with user)

- **Identity**: the client **auto-registers** its local user onto the server
  database (no manual server credentials entered by the user). The provisioned
  user's permissions are then managed by the **server-side admin**.
- **Scope**: knowledge bases, agents, and client-local data pushed to server.

### Server capability constraint (from investigation)

- The server has **no "admin creates user" endpoint**; users only come from
  registration (self_serve / register-by-invite / OIDC).
- The API token grants **TenantRoleAdmin**, but the "add tenant member /
  invite" endpoints are **Owner-gated**, so the API token cannot add a user to
  a tenant directly.
- `POST /auth/register` is gated by `registration_mode` (may be disabled).

Therefore Phase 1 adds a dedicated **Admin-gated provisioning endpoint** on the
server. This is a server-side change; the server must be rebuilt/redeployed.

## 4. Design — three phases

This is a large feature. It is deliberately split so each phase delivers value
and the hardest part (data push) comes last.

### Phase 1 — Auto-provisioning (foundation, unlocks sharing)

The client **auto-registers** its local user onto the server via a new
Admin-gated endpoint, then authenticates as that user (JWT). The server admin
manages the provisioned user's permissions afterwards.

#### Server side (new endpoint)

`POST /api/v1/tenants/:id/client-users` — gated by `g.Admin()` +
`PathTenantMatch` (so the API token provisions into its own tenant).

Request: `{ email, username, password, role? }`
- `password` is generated and owned by the client (so the client can always
  re-authenticate); the server just stores its hash.
- `role` defaults to `contributor` (the role the user gets in tenant `:id`).

Behavior (idempotent):
1. If no user with `email` exists → create one (user + home tenant via
   `userService.Register`, which hashes `password`).
2. If the user exists → ensure the password matches / reset it to `password`.
3. Ensure the user is a member of tenant `:id` with `role` (create or update
   the `tenant_members` row).
4. Return `{ user_id, email, tenant_id, role }`.

This bypasses `registration_mode` (it calls the service directly, not the
gated `/auth/register` handler) and is callable by the Admin API token.

#### Client side (provisioning flow)

1. When a server connection is established, the client provisions its local
   user: generate a stable password (stored encrypted, per server), then call
   `POST /tenants/:id/client-users` with the local user's email/username +
   that password (using the API token).
2. Client logs in as the provisioned user (`POST /auth/login` with email +
   password) → obtains JWT + refresh token.
3. Client stores the linked identity (encrypted at rest, reusing DPAPI):
   `linked_user_id`, `linked_email`, `linked_tenant_id`, `linked_password`,
   `server_jwt`, `server_refresh_token`.
4. Enterprise requests send `Authorization: Bearer <server_jwt>` so the server
   sees the specific user. The API token is kept as a fallback for
   bootstrap/discovery and re-provisioning.
5. On 401, refresh the JWT (`/auth/refresh`); if that fails, re-provision
   (step 1) then re-login.

Data model (extend `enterprise_servers`):
```sql
ALTER TABLE enterprise_servers ADD COLUMN linked_user_id TEXT;
ALTER TABLE enterprise_servers ADD COLUMN linked_email TEXT;
ALTER TABLE enterprise_servers ADD COLUMN linked_tenant_id TEXT;
ALTER TABLE enterprise_servers ADD COLUMN linked_password TEXT;      -- encrypted
ALTER TABLE enterprise_servers ADD COLUMN server_jwt TEXT;           -- encrypted
ALTER TABLE enterprise_servers ADD COLUMN server_refresh_token TEXT; -- encrypted
```

Why this unlocks sharing: the provisioned user is an ordinary server user (a
member of the admin's tenant). The admin manages them with existing endpoints —
`PUT /tenants/:id/members/:user_id` (role), `POST /organizations/:id/invite`
(org membership) — and shares KBs/agents into organizations that include the
user's tenant. The existing organization machinery then works unchanged.

### Phase 2 — Access shared resources (server → client)

With a linked identity, the client pulls what the server has shared with it:

- Fetch the user's organizations and the KBs/agents shared into them
  (`GET /api/v1/organizations/...`, `GET /api/v1/shared-knowledge-bases`,
  `GET /api/v1/shared-agents`).
- Surface these in the client with an enterprise origin badge (reuse
  `ResourceOriginBadge`).
- Chat with shared KBs/agents by proxying to the server with the linked JWT.

Note: shared endpoints strip owner storage metadata; access is capped by the
tenant's organization role (viewer/editor/admin). This is expected.

### Phase 3 — Push local data to server (client → server)

One-way **push** of client-local resources (not full bidirectional sync — that
would require conflict resolution and is out of scope):

- Push a local agent: create the agent on the server via the agent API.
- Push a local KB: create the KB on the server and upload its source documents
  (server then runs its own parsing/embedding pipeline).
- Track push state (pushed / dirty / error) so the user can re-push.

This is the largest phase (document upload + server-side processing), and is
delivered last.

## 5. Non-goals

- Full bidirectional, conflict-resolving sync.
- Syncing chat sessions / messages between client and server.
- Auto-provisioning server accounts without server credentials.
- Offline queueing of pushes (Phase 3 pushes when online; no store-and-forward).

## 6. Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Server JWT expires | enterprise calls 401 | store refresh token; refresh on 401; fall back to API token |
| Linked account lacks org membership | nothing shared visible | admin must invite the linked tenant to an org (documented) |
| KB push = large doc upload + server processing | slow / can fail | per-KB push with progress + error state; retry |
| Two identities (local + server) confuse the user | UX | clear "linked as <email>" indicator in the server panel |
