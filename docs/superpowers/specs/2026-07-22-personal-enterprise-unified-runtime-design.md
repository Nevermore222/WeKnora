# Xelora Personal Enterprise Unified Runtime Design

Status: Approved in design discussion; awaiting written-spec review
Date: 2026-07-22
Target: Xelora Personal for Windows
Branch context: `002-session-workspace-binding`

## 1. Summary

Xelora Personal supports two explicit runtime contexts:

1. **Personal offline context** uses the embedded Go backend, local SQLite,
   local models, local knowledge bases, local agents, and local skills.
2. **Enterprise context** uses an authenticated Xelora server as the sole
   source of business data and execution. The desktop application is another
   client of the same server APIs used by the web application.

Enterprise context does not synchronize, aggregate, or map server resources
into the local database. Users, tenants, memberships, organizations, agents,
knowledge bases, sessions, messages, models, skills, files, and permissions
remain server-owned.

This design supersedes the enterprise-integration portions of
`specs/004-personal-client`, and supersedes the independent-instance,
linked-user, and resource-proxy model in `specs/005-client-server-sync` and
`specs/006-client-user-identity-and-sharing`. The local personal runtime from
004 remains valid.

## 2. Goals

- Make enterprise usage in Xelora Personal behaviorally consistent with the
  web application.
- Show the same server-owned spaces, resources, sessions, sharing state, and
  permissions in both clients.
- Reuse the server's existing account, tenant, organization, invitation,
  sharing, and RBAC model.
- Support multiple configured enterprise servers, with independent accounts
  and credentials per server.
- Preserve a fully separate personal offline context.
- Keep desktop-only capabilities such as system tray, notifications, native
  file selection, downloads, and deep links.

## 3. Non-goals

- Bidirectional synchronization between local and enterprise data.
- Offline copies of enterprise business data.
- Automatic upload or merging of existing personal resources.
- Client-specific user provisioning or administrator-token authentication.
- Client-side authorization decisions that replace server RBAC.
- Arbitrary URL proxying through the desktop process.

## 4. Product Model

The application exposes a runtime-context selector with entries such as:

- Personal Offline
- Company Server A
- Company Server B

Selecting an enterprise server activates the last authenticated account for
that server or presents the server's normal login choices. After login, the
user can select any tenant in their server-provided memberships. Tenant
selection controls the same `X-Tenant-ID` request context used by the web
application.

The effective context key is:

```text
personal

or

enterprise:<server-profile-id>:<user-id>:<tenant-id>
```

All in-memory resource state and non-secret UI preferences are scoped to this
key. A context switch cancels active requests and clears tenant-scoped stores
before loading the destination context.

## 5. Architecture

```text
Xelora Personal (Wails)
|
+-- Shared Vue application
|   +-- RuntimeContextStore
|   +-- Context-aware API client
|   +-- Existing auth, tenant, organization, agent, KB, chat, and settings UI
|
+-- Embedded Go backend
    +-- Personal runtime -> SQLite and local execution
    +-- ServerProfileStore
    +-- CredentialStore -> Windows Credential Manager
    +-- DesktopTransportGateway
    +-- CapabilityHandshake
    +-- DeepLinkHandler

Enterprise request:

Vue standard API call
  -> loopback DesktopTransportGateway
  -> selected enterprise server /api/v1
  -> server database, storage, execution, and RBAC
```

The gateway is a transport and credential boundary, not an enterprise
business layer. It forwards the original API method, path, query, body, and
stream semantics. It does not introduce per-resource proxy contracts.

## 6. Runtime Components

### 6.1 RuntimeContextStore

The store owns the active mode, server profile, account snapshot, tenant, and
capability result. It coordinates transitions in this order:

1. Mark the old context as leaving.
2. Abort HTTP requests, SSE streams, uploads, and downloads.
3. Clear auth, chat, agent, knowledge-base, model, organization, and settings
   stores that contain context-scoped data.
4. Activate the new API context.
5. Restore an enterprise desktop session or initialize the personal runtime.
6. Load identity, memberships, tenant, capabilities, and visible resources.

A response carries the context generation that created its request. The UI
discards a late response when its generation no longer matches the active
context.

### 6.2 ServerProfileStore

Server profiles contain only non-secret connection metadata:

```text
id
name
base_url
allow_insecure_transport
trusted_certificate_fingerprint
last_user_id
last_tenant_id
created_at
updated_at
```

The normalized server origin is immutable during an authenticated session.
Changing it terminates the session and requires authentication against the new
origin.

### 6.3 CredentialStore

Enterprise refresh tokens are stored in Windows Credential Manager, keyed by
server profile and user ID. Access tokens live only in Go process memory.
Passwords are never stored.

The Vue application receives an authenticated-session snapshot containing
user, tenant, memberships, and expiry information, but it does not receive the
access or refresh token. Existing frontend checks that rely on token presence
must instead use the desktop session state in enterprise context.

### 6.4 DesktopTransportGateway

The frontend addresses an enterprise API through a local prefix:

```text
/desktop/remote/<server-profile-id>/api/v1/...
```

The gateway strips the desktop prefix and forwards the original `/api/v1/...`
path to the selected server. It supports JSON, multipart upload, byte-range
download, SSE, cancellation, and backpressure.

The gateway:

- injects the in-memory bearer token;
- forwards `X-Tenant-ID`, locale, request ID, and safe content headers;
- refreshes an expired token once and retries only replay-safe requests;
- preserves server status codes and structured error bodies;
- strips hop-by-hop headers and rejects redirects to another origin;
- accepts only saved, enabled server profiles;
- binds to loopback and requires a per-process random desktop session secret.

Wails injects that desktop session secret into frontend memory during startup;
it is never persisted. Requests without the secret are rejected before proxy
target resolution.

Login and token refresh are gateway-managed operations. The gateway parses the
server login response, stores the refresh token, retains the access token in
memory, and returns a token-free identity snapshot to Vue.

### 6.5 CapabilityHandshake

The server exposes a public, read-only capability response with:

```json
{
  "api_contract_major": 1,
  "api_contract_minor": 0,
  "server_version": "...",
  "features": ["tenant_rbac", "organizations", "shared_resources", "sse_chat"]
}
```

The response contains no deployment secrets or user-specific data, so the
desktop client can validate compatibility before showing the login flow. The
desktop client requires a matching API contract major version. Minor features
are enabled only when advertised. A major mismatch blocks enterprise entry and
identifies whether the client or server must be upgraded.

## 7. Authentication And Identity

Enterprise context uses the server's normal authentication model:

- Account/password login uses the existing server login endpoint through the
  gateway.
- OIDC uses the system browser with PKCE. The callback returns through a
  loopback callback owned by the desktop process or the registered
  `xelora://` protocol, then the gateway completes the desktop session.
- Registration mode, invitation registration, account state, and password
  policy remain server-controlled.
- Logout revokes or discards only the active server session and does not affect
  personal data or other server profiles.

The client does not accept an administrator API token and does not call
`/tenants/:id/client-users`. No linked client identity, generated password, or
synthetic membership is created.

## 8. Tenants, Spaces, And Sharing

After authentication, `/auth/me` and the login session provide the user's home
tenant and memberships. The existing tenant selector chooses the active tenant
and the gateway forwards its ID as `X-Tenant-ID`.

Within the active tenant, the desktop client displays every resource returned
by the same list APIs used by the web client. This includes resources created
by other members when server permissions allow them to be viewed. It does not
grant access to unrelated private resources elsewhere on the server.

Organization/shared-space APIs provide resources shared across tenants. The
desktop client uses the same organization membership, role, shared knowledge
base, shared agent, invitation, approval, and revocation endpoints as the web
client.

Server web share pages may offer an "Open in Xelora Personal" action. The web
page exchanges the existing invitation or share token for a single-use,
short-lived desktop handoff code. The `xelora://` payload identifies the server
origin and carries only that handoff code. The desktop client selects or adds
the server, exchanges the code, and then uses the server's standard preview,
registration, or join flow. Invalid, used, revoked, and expired codes remain
server decisions.

## 9. Enterprise Data Flow

In enterprise context:

- agent, knowledge-base, model, skill, MCP, tenant, member, organization, and
  settings CRUD writes directly to the server;
- session creation returns the server session ID used for all later messages;
- chat streams directly from the server through the transport gateway;
- conversation history is loaded from and stored on the server;
- uploads and generated artifacts use server storage and workspace settings;
- resource IDs remain the original server IDs;
- business responses are kept only in active in-memory UI stores.

There is no `enterprise:<server>:<resource>` identifier, enterprise resource
cache, local-to-remote session map, or local fallback for enterprise writes.

## 10. Error And Offline Behavior

- An unreachable server remains selected but disconnected. The UI does not
  display stale enterprise business data and does not silently fall back to
  personal storage.
- Users may explicitly switch to personal context or another server.
- A `401` triggers one gateway-managed refresh. Refresh failure returns the
  user to that server's login screen.
- A `403` displays a permission error and refreshes identity and memberships.
- A `404` is classified using the server error code as deleted resource,
  inaccessible tenant, or unsupported endpoint.
- An interrupted chat stream queries server session/message state. It resumes
  or polls when generation is still active and offers a non-duplicating retry
  when generation failed.
- Context switches cancel in-flight work. Late responses are discarded using
  the context generation.
- Uploads retry automatically only when the server supports an idempotency key.
- The client never downgrades HTTPS automatically.

HTTP is permitted without an extra warning only for `localhost` and
`127.0.0.1`. Private-LAN HTTP or a self-signed certificate requires explicit
per-profile approval. For a self-signed certificate, the approved fingerprint
is pinned; a changed certificate blocks reconnection until re-approved.

## 11. Security Boundaries

- The gateway listens only on loopback and validates a per-process session
  secret on every desktop request.
- A request can target only the origin in its saved server profile.
- Redirects to a different origin, DNS rebinding to disallowed addresses, and
  proxying arbitrary URLs are rejected.
- Authorization, cookie, proxy, forwarding, and hop-by-hop headers supplied by
  page code are removed or rebuilt by the gateway.
- Passwords and tokens are redacted from logs, diagnostics, and crash reports.
- Enterprise authorization is enforced by server middleware. Frontend role
  checks control presentation only.
- Deep links require user confirmation before adding a new server and never
  carry passwords or bearer tokens.

## 12. Migration

Existing `enterprise_servers` rows migrate only their name, normalized base
URL, timestamps, and last-use metadata into server profiles. Administrator API
tokens, linked passwords, linked JWTs, linked refresh tokens, linked user IDs,
and linked tenant IDs are not imported into the new credential store.

After the user successfully authenticates with the new model, obsolete secrets
for that profile are securely removed from the legacy store. Migration never
uploads or changes personal resources.

The following enterprise-specific frontend and backend concepts leave the
active code path and are removed after migration compatibility is verified:

- linked-user provisioning;
- resource aggregation and cache;
- per-resource enterprise proxy endpoints;
- enterprise resource ID prefixes;
- local/server session mapping;
- enterprise-only resource selector branches and stores.

## 13. Delivery Boundaries

Implementation is divided into independently reversible increments:

1. Introduce runtime context, server profiles, the capability endpoint, and the
   loopback transport gateway without changing the existing personal path.
2. Add gateway-managed enterprise login, credential storage, tenant selection,
   and context isolation.
3. Route the existing web resource and chat APIs through enterprise context,
   then remove special enterprise IDs, stores, and session mapping from the
   active path.
4. Add legacy profile migration, secure secret cleanup, OIDC, and desktop
   handoff deep links.
5. Remove obsolete linked-user and per-resource proxy code after cross-client,
   migration, and personal-mode regression suites pass.

Each increment must leave personal mode usable. Until increment 5, the legacy
path may remain behind a disabled compatibility flag, but new enterprise
sessions never use it.

## 14. Verification

### 14.1 Cross-client consistency

- Create and edit an agent, knowledge base, and session in the web client;
  verify identical IDs, content, creator, and timestamps in desktop.
- Repeat desktop to web.
- Alternate messages in one session between web and desktop and verify one
  ordered server history.
- Verify shared-resource visibility and revocation in both clients.

### 14.2 Identity And RBAC

- Test two users, two tenants, one organization/shared space, and Viewer,
  Contributor, Admin, and Owner roles.
- Compare visible actions and server responses in web and desktop.
- Verify membership add, role change, suspension, removal, invitation, and
  tenant switching take effect without a desktop restart.

### 14.3 Context Isolation

- Switch among personal context, two servers, two users, and multiple tenants.
- Verify no agent, knowledge base, model, session, UI selection, request, or
  stream crosses context boundaries.
- Verify a late response from the previous context is discarded.

### 14.4 Gateway And Failure Recovery

- Cover JSON, multipart, range download, SSE, cancellation, timeout, token
  refresh, certificate validation, redirect rejection, and large payloads.
- Test disconnect during chat and upload, permission revocation, tenant exit,
  deleted resources, expired invitations, and API contract mismatch.
- Verify the gateway cannot reach an unsaved origin or be used as an open
  proxy.

### 14.5 Release Regression

- Run the complete personal offline regression suite.
- Run one shared enterprise end-to-end suite against both the browser client
  and the packaged Windows client.
- Verify clean install, upgrade install, legacy profile migration, Credential
  Manager cleanup, protocol registration, and deep-link launch.

## 15. Acceptance Criteria

The design is complete when all of the following are true:

1. The same enterprise account and active tenant expose the same authorized
   resources and sessions in web and desktop.
2. A write in either client is visible in the other without synchronization or
   import.
3. Server RBAC and sharing decisions produce equivalent behavior in both
   clients.
4. Enterprise business data is never persisted in the personal SQLite data
   model.
5. Switching context cannot leak requests, credentials, selections, or results.
6. Enterprise credentials are absent from frontend storage and ordinary local
   database fields.
7. Personal offline behavior remains functional and independent.
