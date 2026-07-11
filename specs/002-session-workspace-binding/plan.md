# Implementation Plan: Session Workspace Binding For New Chats

**Branch**: `002-session-workspace-binding` | **Date**: 2026-07-10 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-session-workspace-binding/spec.md`

## Summary

Add an optional workspace binding to the new-chat flow, persist it as durable session state, and route file-producing runtime/artifact behavior through that binding by default. The implementation keeps legacy sessions unbound, keeps permission validation explicit, and shifts default output ownership from skill-private working folders to a conversation-owned workspace contract.

## Technical Context

**Language/Version**: Go 1.26 backend, Vue 3 + TypeScript frontend

**Primary Dependencies**: Gin-style HTTP handlers, GORM/PostgreSQL persistence, Pinia state management, existing executor gateway and artifact runtime

**Storage**: PostgreSQL `sessions` table plus existing filesystem/object-storage-backed artifact paths

**Testing**: Go unit/integration tests, frontend component/state tests where practical, manual end-to-end validation through the web UI

**Target Platform**: Dockerized web application running on Linux/Docker Desktop

**Project Type**: Web application with backend + frontend

**Performance Goals**: Preserve current session creation latency expectations; workspace binding lookup must add only one lightweight validation/read step per file-producing turn

**Constraints**: Must remain backward compatible for legacy sessions, must not silently fall back to hidden skill workspaces for bound conversations, must enforce workspace boundary checks before writes

**Scale/Scope**: One default workspace per conversation in v1; applies to new-chat creation, session hydration, runtime execution routing, and artifact attribution

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

No project constitution has been initialized at `.specify/memory/constitution.md`, so there are no constitution gates to enforce yet.

Working rules applied for this plan:

- Reuse current session, executor, and artifact modules instead of introducing a parallel workspace system.
- Make the smallest contract change that supports durable binding and runtime routing.
- Preserve compatibility for existing sessions and existing non-file chat flows.

## Project Structure

### Documentation (this feature)

```text
specs/002-session-workspace-binding/
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/
|   `-- session-workspace-binding.openapi.yaml
`-- tasks.md
```

### Source Code (repository root)

```text
internal/
|-- application/
|   |-- repository/
|   `-- service/
|-- executor/
|-- handler/
|   `-- session/
`-- types/

frontend/src/
|-- api/chat/
|-- stores/
`-- views/
    |-- chat/
    `-- creatChat/
```

**Structure Decision**: Keep this feature inside existing session/executor/frontend modules. Backend owns durable binding state and validation, executor consumes the resolved workspace contract for output routing, and frontend only handles bind/select/display flows.

## Phase 0: Research

See [research.md](./research.md).

Primary decisions:

1. Store workspace binding as durable session state, not transient UI memo.
2. Keep one stable conversation-to-workspace contract that both session APIs and runtime code understand.
3. Validate binding on use, not only on create, so access loss and archived workspaces are caught before writes.
4. Treat legacy sessions as explicitly unbound instead of auto-migrating or guessing bindings.

## Phase 1: Design

See:

- [data-model.md](./data-model.md)
- [contracts/session-workspace-binding.openapi.yaml](./contracts/session-workspace-binding.openapi.yaml)
- [quickstart.md](./quickstart.md)

Design outcomes:

1. Session APIs gain an optional `workspace_binding` payload and return binding status on reads.
2. Session domain gains a durable workspace-binding model with validation state.
3. Runtime output routing resolves a conversation output root from the session binding and uses it for artifact ownership.
4. Failure modes become explicit: invalid binding, inaccessible binding, unsafe path escape, and unbound conversation.

## Implementation Strategy

### Backend

1. Extend session types and persistence to store `workspace_binding`.
2. Update create/get/update session flows to validate, persist, and return binding state.
3. Add a reusable service for resolving effective conversation output roots.
4. Update executor/artifact paths so default file-producing jobs use the bound workspace instead of a skill-private root when a valid binding exists.

### Frontend

1. Add workspace selection to the new-chat flow.
2. Send binding payload on session creation.
3. Hydrate binding state when reopening sessions and expose bound/unbound status in the chat context.
4. Surface clear user feedback when a binding is invalid or missing for file-producing actions.

### Verification

1. New chat with workspace binding creates a durable session binding.
2. Reopened chat preserves the same binding.
3. File-producing turn writes/registers artifacts under the bound workspace.
4. Invalid or inaccessible workspace blocks unsafe writes with a visible error.
5. Legacy sessions still work as unbound conversations.

## Complexity Tracking

No constitution violations or exceptional complexity exemptions are required for this feature.
