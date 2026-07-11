# Tasks: Session Workspace Binding For New Chats

**Input**: Design documents from `F:\Docker\WeKnora\Xelora\specs\002-session-workspace-binding\`

**Prerequisites**: `plan.md` (required), `spec.md` (required), `research.md`, `data-model.md`, `contracts/session-workspace-binding.openapi.yaml`, `quickstart.md`

**Tests**: The feature specification requires independent validation scenarios, but it does not require a TDD-first workflow. This task list focuses on implementation tasks plus targeted verification updates in existing Go tests and quickstart validation.

**Organization**: Tasks are grouped by user story so each story can be implemented and reviewed independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel when the task touches different files and does not depend on incomplete work
- **[Story]**: User story label for traceability (`[US1]`, `[US2]`, `[US3]`)
- Every task includes the exact file path to update

## Phase 1: Setup (Shared Contract Entry Points)

**Purpose**: Align the active feature docs and API reference entry points before code changes begin.

- [x] T001 Add the workspace-binding feature links to `docs/customizations/README-dev.md`
- [x] T002 [P] Add the session workspace binding API notes to `docs/api/session.md`
- [x] T003 [P] Sync the feature handoff note for workspace-bound conversations in `docs/customizations/TASKS.md`

---

## Phase 2: Foundational (Blocking Session And Runtime Primitives)

**Purpose**: Create the shared backend primitives that all user stories depend on.

**Critical**: No user-story work should start until these tasks are complete.

- [x] T004 Extend session workspace-binding domain types in `internal/types/session.go`
- [x] T005 [P] Extend session create/update request payload types for `workspace_binding` in `internal/handler/session/types.go`
- [x] T006 [P] Add repository support for persisting and reading `workspace_binding` in `internal/application/repository/session.go`
- [x] T007 Add shared workspace-binding validation and hydration helpers in `internal/application/service/session.go`
- [x] T008 Define executor-side conversation output context types in `internal/executor/types.go`
- [x] T009 Document the stable create/get/update binding contract in `specs/002-session-workspace-binding/contracts/session-workspace-binding.openapi.yaml`

**Checkpoint**: Session state and runtime types can now carry a durable workspace binding end to end.

---

## Phase 3: User Story 1 - Bind A Workspace When Starting A New Chat (Priority: P1) MVP

**Goal**: Let users choose a workspace during new chat creation and persist that binding as durable session state.

**Independent Test**: A user can create a new chat with a selected workspace, reopen the session, and see the same binding returned by the session API and rendered in the chat context.

### Implementation for User Story 1

- [x] T010 [US1] Accept and validate `workspace_binding` in session create and update handlers in `internal/handler/session/handler.go`
- [x] T011 [P] [US1] Persist and return workspace binding state from session service flows in `internal/application/service/session.go`
- [x] T012 [P] [US1] Update session create/get API client payloads for `workspace_binding` in `frontend/src/api/chat/index.ts`
- [x] T013 [US1] Add workspace-binding state to the settings/session hydration store in `frontend/src/stores/settings.ts`
- [x] T014 [US1] Add workspace selection and create-session payload wiring in `frontend/src/views/creatChat/creatChat.vue`
- [x] T015 [US1] Render bound or unbound session workspace context in `frontend/src/views/chat/index.vue`
- [x] T016 [US1] Update the session API documentation examples for workspace-bound creation in `docs/api/session.md`

**Checkpoint**: New chats can be bound to one workspace and that binding survives session reopen.

---

## Phase 4: User Story 2 - Keep Conversation Outputs Inside The Bound Workspace (Priority: P2)

**Goal**: Route default file-producing execution and artifact ownership through the bound conversation workspace instead of a hidden skill-private root.

**Independent Test**: After creating a workspace-bound conversation, a file-producing request writes or registers artifacts against the bound workspace, and reopened conversations keep using the same output root by default.

### Implementation for User Story 2

- [x] T017 [US2] Resolve conversation output context from session binding during runtime preparation in `internal/executor/gateway.go`
- [x] T018 [P] [US2] Extend workspace and artifact metadata to carry bound workspace ownership in `internal/executor/types.go`
- [x] T019 [P] [US2] Pass conversation workspace binding through skill execution requests in `internal/agent/tools/skill_execute.go`
- [x] T020 [US2] Update skill manager expectations for conversation-owned outputs in `internal/agent/skills/manager.go`
- [x] T021 [US2] Route default executor workspace roots to the bound workspace when available in `internal/executor/gateway.go`
- [x] T022 [US2] Update executor seam coverage for bound workspace routing and legacy unbound sessions in `internal/executor/gateway_test.go`
- [x] T023 [US2] Document the bound-workspace output contract and artifact expectations in `specs/002-session-workspace-binding/quickstart.md`

**Checkpoint**: Bound conversations now default file output and artifact attribution to the user-selected workspace.

---

## Phase 5: User Story 3 - Protect Workspace Boundaries And Failure Handling (Priority: P3)

**Goal**: Block unsafe writes, surface invalid bindings clearly, and preserve explicit recovery behavior for unbound or inaccessible conversations.

**Independent Test**: When a workspace is inaccessible, deleted, or escapes boundary rules, the system blocks the write and returns a clear user-visible failure instead of silently writing elsewhere.

### Implementation for User Story 3

- [x] T024 [US3] Enforce workspace access and boundary failure states in session binding validation paths in `internal/application/service/session.go`
- [x] T025 [P] [US3] Reject executor path escapes and invalid default output roots in `internal/executor/gateway.go`
- [x] T026 [P] [US3] Add invalid-binding and path-escape coverage to executor tests in `internal/executor/gateway_test.go`
- [x] T027 [US3] Return user-visible binding failure details from session handlers in `internal/handler/session/handler.go`
- [x] T028 [US3] Surface invalid/unbound workspace recovery messaging in `frontend/src/views/chat/index.vue`
- [x] T029 [US3] Preserve explicit unbound-session behavior in store hydration and create-chat flows in `frontend/src/stores/settings.ts`
- [x] T030 [US3] Update failure and recovery examples for invalid bindings in `docs/api/session.md`

**Checkpoint**: Unsafe writes are blocked, invalid bindings are visible, and unbound behavior is explicit rather than hidden.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish the feature handoff cleanly and validate cross-artifact consistency.

- [x] T031 [P] Refresh the feature summary and rollout note in `README.md`
- [x] T032 [P] Reconcile `plan.md`, `tasks.md`, and `quickstart.md` wording in `specs/002-session-workspace-binding/plan.md`
- [x] T033 Perform a final consistency pass across `AGENTS.md`, `docs/api/session.md`, and `specs/002-session-workspace-binding/tasks.md`
- [x] T034 Run the manual validation flow and record the final expected checks in `specs/002-session-workspace-binding/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup**: No dependencies
- **Phase 2: Foundational**: Depends on Phase 1 and blocks all user stories
- **Phase 3: US1**: Depends on Phase 2
- **Phase 4: US2**: Depends on US1 because runtime output routing needs durable bound session state
- **Phase 5: US3**: Depends on US1 and US2 because failure handling must enforce the finalized binding and output contract
- **Phase 6: Polish**: Depends on all earlier phases

### User Story Dependencies

- **US1 (P1)**: Starts first and establishes the durable conversation workspace binding contract
- **US2 (P2)**: Builds on US1 by routing runtime outputs and artifact ownership through that binding
- **US3 (P3)**: Builds on US1 and US2 by enforcing validation, boundary checks, and recovery behavior

### Parallel Opportunities

- `T001`, `T002`, and `T003` can proceed in parallel at setup time
- `T005` and `T006` can run in parallel after `T004`
- `T011` and `T012` can run in parallel after handler and domain primitives are in place
- `T018` and `T019` can run in parallel after `T017`
- `T025` and `T026` can run in parallel after the initial boundary rules are defined
- `T031` and `T032` can run in parallel during polish

---

## Parallel Example: User Story 1

```text
Task: "Persist and return workspace binding state from session service flows in internal/application/service/session.go"
Task: "Update session create/get API client payloads for workspace_binding in frontend/src/api/chat/index.ts"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. Stop and validate that a new chat can bind, persist, and rehydrate one workspace cleanly

### Incremental Delivery

1. Establish durable session binding primitives
2. Deliver the new-chat binding flow and session hydration
3. Route runtime outputs through the bound workspace
4. Add failure handling and boundary enforcement
5. Finish with docs and validation cleanup

### Suggested MVP Scope

- Phase 1
- Phase 2
- Phase 3 (US1)

---

## Notes

- Keep `workspace_binding` separate from `last_request_state`; they serve different durability guarantees.
- Do not silently fall back to skill-private output roots for bound conversations once US2 is complete.
- Preserve legacy sessions as explicitly unbound unless the user binds a workspace later through the supported contract.
