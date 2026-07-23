# 005 — Client–Server Sync: Implementation Plan

Status: Draft (pending user approval)
Depends on: 004-personal-client

## Phasing

Deliver in three phases. Each phase is independently valuable and shippable.
Phase 1 is the foundation and must land first.

---

## Phase 1 — Auto-provisioning (foundation) — COMPLETE

Goal: client auto-registers its local user onto the server (new Admin-gated
endpoint); the client then authenticates as that user (JWT) so the server can
share with them and the admin can manage their permissions.

### Server side (requires server rebuild/reploy)

- [ ] P1-S1 Add a service method to provision a client user idempotently:
  create user (via `userService.Register`) if absent, else reset password;
  then ensure a `tenant_members` row for `(user, tenantID)` with the given
  role (default `contributor`). Lives in the user/tenant service layer.
- [ ] P1-S2 Add handler `ProvisionClientUser` in `internal/handler/tenant_member.go`
  (or a new `client_user.go`): parse `{email, username, password, role?}`,
  call the service, return `{user_id, email, tenant_id, role}`.
- [ ] P1-S3 Register route `POST /tenants/:id/client-users` in
  `internal/router/router.go`, gated by `g.Admin()` + `g.PathTenantMatch()`.

### Client side

- [ ] P1-C1 Extend `enterprise.ServerConfig` with linked-identity fields
  (`linked_user_id`, `linked_email`, `linked_tenant_id`, `linked_password`,
  `server_jwt`, `server_refresh_token`). GORM auto-migrate adds the columns.
- [ ] P1-C2 `store.go`: persist linked identity; encrypt `linked_password`,
  `server_jwt`, `server_refresh_token` via existing `EncryptToken` (DPAPI);
  decrypt on read.
- [ ] P1-C3 `connector.go`: add `ProvisionUser(serverID, localEmail,
  localUsername)` — generate/reuse a per-server password, call
  `POST /tenants/:id/client-users` with the API token, then
  `POST /auth/login` to obtain JWT + refresh token; store everything.
- [ ] P1-C4 `proxy.go`: when a linked JWT exists, send
  `Authorization: Bearer <jwt>`; keep `X-API-Key` as fallback. On 401, try
  `/auth/refresh`; if that fails, re-provision (P1-C3) then retry once.
- [ ] P1-C5 `handler/enterprise.go`: trigger provisioning on connect; expose
  linked status (`linked_email`) in `GET /enterprise/servers` and `.../status`.

### Frontend

- [ ] P1-C6 `EnterpriseServerManager.vue`: show "已关联为 <email>" when a
  server has a linked user (provisioning is automatic on connect, no manual
  credentials). Optionally a "重新配置 / Re-provision" action.
- [ ] P1-C7 i18n strings (zh-CN + en-US) for the linked-status UI.

### Verification (Phase 1)

- Rebuild + redeploy the server with the new endpoint.
- Connect the client; confirm it provisions the local user (user appears in
  the server's tenant members).
- Confirm `GET /auth/me` through the proxy returns the provisioned user.
- On the server web UI, confirm the admin can change the user's role and share
  a space with the user's tenant.

---

## Phase 2 — Access shared resources (server → client) — COMPLETE

Goal: client pulls and uses KBs/agents the server has shared with the linked
user.

- [ ] P2-1 `connector.go`: fetch the linked user's organizations + shared
  KBs/agents (`/organizations`, `/shared-knowledge-bases`, `/shared-agents`)
  with the linked JWT; merge into `ServerCapabilities`.
- [ ] P2-2 `handler/enterprise.go`: surface shared resources in
  `GET /enterprise/resources` (mark `origin=enterprise`, `shared=true`).
- [ ] P2-3 Frontend: show shared KBs/agents in the selectors with an
  enterprise badge; route chat for a shared resource through the proxy.
- [ ] P2-4 i18n for shared-resource labels.

### Verification (Phase 2)

- On the server, share a KB into an org that includes the linked tenant.
- In the client, confirm the shared KB appears and can be used in chat.

---

## Phase 3 — Push local data to server (client → server)

Goal: one-way push of client-local agents and KBs (with documents) to the
server.

- [ ] P3-1 `connector.go`: `PushAgent(serverID, localAgentID)` — create the
  agent on the server via the agent API.
- [ ] P3-2 `connector.go`: `PushKnowledgeBase(serverID, localKBID)` — create
  the KB on the server, upload its source documents (server processes them).
- [ ] P3-3 Track push state per resource (pushed / dirty / error) in a local
  table `enterprise_push_state`.
- [ ] P3-4 `handler/enterprise.go`: expose push endpoints + status.
- [ ] P3-5 Frontend: a "上传到服务器 / Push to server" action on local
  KBs/agents with progress + error feedback.
- [ ] P3-6 i18n for push UI.

### Verification (Phase 3)

- Push a local KB (with a document) to the server; confirm it appears on the
  server and finishes processing; confirm re-push updates it.

---

## Effort note

Phase 1 and Phase 2 are moderate. Phase 3 (especially KB push with document
upload + server-side parsing) is the largest and riskiest; it is deliberately
last. Recommend landing Phase 1 first and validating the sharing flow end to
end before committing to Phase 2/3.
