# Research: Session Workspace Binding For New Chats

## Decision: Persist workspace binding as first-class session state

**Rationale**: The current `SessionLastRequestState` only memoizes input-bar choices such as agent, model, selected knowledge bases, and web search. It is explicitly best-effort UI state and is stored through `agent_config` updates. A workspace binding must survive reopen, cross-device continuation, and runtime execution routing, so it needs to be part of the durable session contract rather than transient UI restoration state.

**Alternatives considered**:

- Reuse `last_request_state`: rejected because it is not semantically durable product state and can be absent for older sessions.
- Store only in frontend local storage: rejected because it fails cross-device continuity and cannot drive backend artifact routing.

## Decision: Keep one conversation-level workspace binding in v1

**Rationale**: The feature spec only requires one default workspace per conversation. Supporting multiple roots, per-turn overrides, or per-skill workspaces would expand scope sharply and complicate validation, UI, and artifact ownership without being necessary for the first useful release.

**Alternatives considered**:

- Multiple bound workspaces per conversation: rejected as premature complexity.
- Per-message workspace binding: rejected because it breaks the simple mental model of "this chat writes here by default."

## Decision: Validate workspace accessibility both at bind time and at write time

**Rationale**: A workspace can become invalid after session creation because of deletion, archival, or permission loss. Bind-time validation prevents obviously bad sessions from being created, while write-time validation prevents unsafe or confusing output behavior later in the conversation.

**Alternatives considered**:

- Validate only when creating the session: rejected because permissions and workspace lifecycle can change later.
- Validate only when executing a file-producing action: rejected because it delays obvious user feedback and makes the create flow less trustworthy.

## Decision: Route default runtime outputs through a resolved conversation output root

**Rationale**: The current executor workspace record is effectively skill-centric, with `buildWorkspaceRecord` deriving `RootPath` from a skill base path and `buildWorkspaceID` combining session id and skill name. That is useful for isolated tool execution, but it does not satisfy user-visible workspace ownership. The new feature should introduce a resolved conversation output root that runtime code can consume consistently when a valid binding exists.

**Alternatives considered**:

- Replace the executor workspace model entirely: rejected because the current runtime still needs skill working directories and provider metadata.
- Keep skill-private roots and only copy artifacts later: rejected because it weakens the contract and makes failures harder to reason about.

## Decision: Keep legacy sessions explicitly unbound

**Rationale**: Existing sessions were created before this feature and do not contain trustworthy workspace information. Guessing a workspace from prior skill paths or user history would be brittle and potentially unsafe. Explicitly unbound legacy behavior is safer and easier to explain.

**Alternatives considered**:

- Auto-migrate old sessions to a default workspace: rejected because it can misroute future outputs.
- Hide the distinction from users: rejected because the product must clearly signal whether file output has a real workspace destination.

## Decision: Make session API the source of truth for workspace binding status

**Rationale**: The frontend new-chat page, chat page hydration, runtime orchestration, and future artifact UX all need the same answer about whether a conversation is bound, unbound, invalid, or access-blocked. Centralizing this in the session API avoids duplicated inference logic in the web client.

**Alternatives considered**:

- Let frontend infer binding state from partial workspace metadata: rejected because the backend owns permission and lifecycle validation.
- Create a completely separate binding API surface first: rejected because create/get session flows already form the natural user entry point.
