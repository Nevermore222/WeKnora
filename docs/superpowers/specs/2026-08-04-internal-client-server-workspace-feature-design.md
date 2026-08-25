# Internal Client Server Workspace Feature Design

**Date:** 2026-08-04

## Decision

The first company-internal desktop client uses the enterprise server runtime.
It does not use Personal local-folder binding. Server workspaces, the existing
Xelora Agent loop, Controlled Docker, preloaded Skills, jobs, artifacts, and
SSE remain the execution path.

The client workspace feature is controlled by a server-issued feature switch.
The first client release hides the feature by default but does not remove the
server API, data model, executor path, or existing bindings.

The switch controls client visibility and new-client entry. It is not a
security boundary. The server continues to enforce workspace access, tenant
RBAC, user capabilities, Skill policy, path confinement, and executor policy.

## Goals

- Ship an internal client that connects to the company Xelora server without
  asking users to configure a server URL.
- Preserve the existing client registration and linked-identity flow.
- Keep server-controlled account capabilities, including knowledge-base and
  agent creation permissions.
- Allow the server workspace feature to be enabled later without rebuilding
  the workspace backend.
- Reuse the existing server workspace binding contract based on `workspace_id`.
- Execute preloaded Skills in Controlled Docker and write approved output files
  into the bound server workspace.
- Preserve the existing Go Agent loop and Progressive RAG behavior.

## Non-goals

- No Personal local-folder binding in the internal client release.
- No client-side filesystem workspace or local sandbox execution.
- No replacement of the Go Agent loop with Pi.
- No new client-side authorization model.
- No arbitrary script upload or arbitrary script execution from the client.
- No deletion or migration of existing session workspace bindings.

## Runtime Architecture

```text
Internal Xelora Client
  -> packaged server URL or single mDNS discovery result
    -> desktop transport / enterprise connector
      -> Xelora Server
        -> authenticated client identity and server capabilities
          -> Go Agent Loop and Progressive RAG
            -> session workspace binding by workspace_id
              -> Executor Gateway
                -> Controlled Docker
                  -> preloaded Skill
                    -> server workspace files and Artifact records
```

The client is a transport and presentation surface. The server owns business
state, permissions, workspace identity, Skill resolution, execution policy,
artifact registration, and the user-visible event stream.

## Server Feature Switch

Use a server-managed setting named `client.workspace_binding_enabled`.

The initial default is `false`. The setting is administered remotely by a
system administrator and is included in the server capability response sent to
the enterprise client after connection or capability refresh.

The setting has presentation and entry semantics only:

| Server setting | New client entry | Existing bindings | Server executor |
| --- | --- | --- | --- |
| `false` | Hidden; no new binding is sent | Preserved | Unchanged |
| `true` | Workspace selector is available | Preserved | Unchanged |

When the setting is `false`, the client does not render the workspace selector,
binding status bar, or workspace file browser. It also omits
`workspace_binding` from new-session requests. The server does not reject old
bound sessions solely because the client feature is hidden.

When the setting is `true`, the client renders the existing server-workspace
selection flow and sends the selected `workspace_id`. The server validates the
binding before persisting it.

The setting must not be confused with `can_bind_model`. Model binding and
workspace binding are different capabilities. The first release uses the
existing workspace membership/access check rather than adding a new
per-user `can_bind_workspace` flag.

Changes to the setting should use the existing system-settings authorization
and audit path. A capability response that cannot be read fails closed for the
workspace UI; it must not disable unrelated chat, agent, knowledge-base, or
shared-resource features.

## Client Startup And Default Server

The internal client has no editable server configuration screen.

Startup behavior:

1. Read the company server origin packaged with the client build.
2. Run the existing server health and capability handshake.
3. If the packaged origin is unavailable, use the existing mDNS discovery path
   only when it returns exactly one compatible Xelora server.
4. If no unambiguous server is available, show a connection failure state and
   do not silently connect to `localhost` or an arbitrary discovered server.
5. On success, continue through the existing client registration, linked
   identity, JWT, tenant, membership, shared-resource, and capability flow.

The packaged server origin is deployment metadata, not a user preference. A
server-address change is delivered through a new internal client package or a
controlled deployment update.

## Authorization Model

The server remains the source of truth.

Existing account capabilities remain unchanged:

- `can_create_knowledge_base` controls knowledge-base creation.
- `can_create_agent` controls agent creation.
- `can_bind_model` controls model management.

Organization membership and shared-resource permissions continue to determine
which knowledge bases and agents the user can view, query, or edit. A server
workspace can be selected only when the authenticated user has access to it.

The client may hide controls based on the capability response, but every
protected server route continues to enforce RBAC and capability middleware.
Tampering with the client request cannot grant access.

## Session And Skill Data Flow

With the feature enabled:

1. The client requests the accessible server workspaces.
2. The user selects one workspace.
3. The client creates a session with `workspace_binding.workspace_id`.
4. The server validates access and persists the durable session binding.
5. The Go Agent loop performs the normal Progressive RAG and knowledge-base
   tool flow.
6. If the agent invokes `execute_skill_script`, Go resolves the Skill through
   the existing Skill Manager.
7. The Executor Gateway resolves the session binding and requires a valid
   bound output context.
8. The selected server provider is Controlled Docker. The client cannot select
   a provider or supply an absolute host path.
9. The gateway stages approved inputs, fixes the working directory to the
   bound server workspace, and runs the preloaded Skill.
10. The gateway snapshots the workspace before and after execution, registers
    changed files as Artifacts, records the Job result, and emits the existing
    tool and answer events through SSE.

The Skill path accepts only Skills already resolved and prepared by Xelora. A
model or client argument cannot turn it into an arbitrary script runner.

With the feature disabled, new sessions remain unbound. Normal chat and
knowledge-base retrieval continue. A file-producing Skill that requires a
workspace returns the existing `workspace_required` failure instead of writing
to a hidden fallback directory.

Existing bound sessions retain their durable binding and continue to use the
server executor. Hiding the client UI does not clear the binding or invalidate
its output contract.

## Client Components

The implementation should preserve the existing component behavior behind a
feature predicate rather than fork the workspace flow:

- The new-chat view renders `WorkspaceSelector` only when the server feature is
  enabled and the runtime is an enterprise server context.
- The new-chat request sends `workspace_id` only when the feature is enabled
  and a workspace was selected.
- The chat view renders binding status, recovery, and the file browser only
  when the feature is enabled, while still hydrating old session data for
  artifact and execution compatibility.
- Personal `root_path` selection remains outside the internal client release.
- Existing knowledge-base, agent, organization, shared-resource, and settings
  components remain available subject to server capabilities.

The client feature state should be refreshed on initial connection, login,
reconnect, and explicit capability refresh. It does not need real-time push in
the first release.

## Failure Behavior

| Condition | Required behavior |
| --- | --- |
| Feature switch is false | Hide workspace UI; keep server APIs and old bindings |
| Capability response unavailable | Hide workspace UI; keep unrelated features available |
| User has no workspace access | Show no selectable workspaces; server still enforces access |
| Forged workspace ID | Server rejects session binding with authorization error |
| Unbound session invokes file Skill | Return `workspace_required`; no hidden fallback write |
| Binding invalid or access revoked | Block execution and return binding error |
| Skill is not preloaded or not allowed | Reject before execution |
| Workspace path escapes boundary | Reject before provider execution |
| Controlled Docker unavailable | Return runtime failure; never execute locally |
| Skill exits unsuccessfully | Preserve stdout/stderr, mark Job failed, and emit failure events |

## Verification Plan

Backend tests must cover:

- capability response for the feature switch enabled and disabled;
- administrator-only switch updates and audit records;
- session binding access checks;
- existing bound sessions after the switch is disabled;
- unbound Skill failure behavior;
- Controlled Docker workspace mounting and path confinement;
- preloaded Skill output and Artifact registration;
- rejection of forged workspace IDs and arbitrary script paths.

Frontend tests must cover:

- workspace controls hidden when the server switch is false;
- workspace controls restored when it is true;
- no binding field in new-session payloads while hidden;
- `workspace_id` included when enabled and selected;
- account capability-driven visibility for knowledge-base and agent creation;
- graceful handling of missing capability responses.

The release smoke test must run against the internal server runtime:

1. Install the client with the packaged server origin.
2. Confirm automatic connection and existing client registration.
3. Confirm normal knowledge-base retrieval and SSE answer behavior.
4. Confirm workspace controls are hidden with the default switch value.
5. Enable the switch as an administrator and refresh the client capabilities.
6. Bind a server workspace and run a preloaded file-producing Skill.
7. Confirm the file and Artifact are inside the server workspace.
8. Disable the switch and confirm existing bindings and server data remain.
9. Confirm an unbound new session still supports normal chat but cannot silently
   write a Skill output.

## Rollout Order

1. Add the server-managed feature response and default-off setting.
2. Add client capability state and conditional rendering around the existing
   server workspace components.
3. Verify default internal client connection and existing registration flow.
4. Run server workspace and preloaded Skill output regression tests.
5. Package the internal client with the company server origin.
6. Enable the server switch only after the end-to-end smoke test passes.

Pi integration is explicitly deferred for this release. If introduced later,
it must sit behind the server Executor Gateway or another Xelora-owned provider
boundary and preserve the same workspace, authorization, Skill, Job, Artifact,
and SSE contracts.
