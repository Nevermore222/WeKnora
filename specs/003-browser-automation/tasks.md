# Tasks: Browser Automation Provider Path

**Input**: Design documents from `F:\Docker\WeKnora\Xelora\specs\003-browser-automation\`

**Prerequisites**: `plan.md` (required), `spec.md` (required), `research.md`, `data-model.md`

**Tests**: The feature specification requires independent validation scenarios. This task list includes targeted Go tests for the gateway browser task path and a manual smoke script.

**Organization**: Tasks are grouped by user story so each story can be implemented and reviewed independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel when the task touches different files and does not depend on incomplete work
- **[Story]**: User story label for traceability (`[US1]`, `[US2]`, `[US3]`)
- Every task includes the exact file path to update

## Phase 1: Setup (Shared Contract Entry Points)

**Purpose**: Align the feature docs and reference entry points before code changes begin.

- [ ] T001 Add the browser automation feature links to `docs/customizations/README-dev.md`
- [ ] T002 [P] Add browser automation API notes to `docs/api/browser.md`
- [ ] T003 [P] Update the shared progress board for T-012 in `docs/customizations/TASKS.md`

---

## Phase 2: Foundational (Blocking Browser Provider Primitives)

**Purpose**: Create the shared backend primitives that all user stories depend on.

**Critical**: No user-story work should start until these tasks are complete.

- [x] T004 Define `BrowserJobRequest`, `BrowserJob`, `BrowserTaskResult`, and `BrowserProvider` interface in `internal/executor/browser_types.go`
- [x] T005 [P] Add browser provider constants and env-var loading in `internal/executor/browser_provider.go`
- [x] T006 Add `RunBrowserTaskJob` to the gateway with workspace resolution, boundary checks, and artifact detection in `internal/executor/gateway.go`
- [x] T007 [P] Document the browser agent tool contract in `specs/003-browser-automation/contracts/browser-automation.openapi.yaml`

**Checkpoint**: The gateway can accept browser task requests, resolve workspace bindings, and return registered artifacts.

---

## Phase 3: User Story 1 - Launch A Browser Task From A Conversation (Priority: P1) MVP

**Goal**: Let an agent dispatch a browser navigation task that captures a screenshot or page content and registers it as an artifact.

**Independent Test**: An agent-enabled conversation can trigger a browser navigation task and the system returns at least one registered artifact downloadable from the chat context.

### Implementation for User Story 1

- [x] T008 [US1] Implement `ControlledDockerBrowserProvider` in `internal/executor/browser_provider.go`
- [x] T009 [US1] Create the `browser-snapshot` skill with `SKILL.md` in `skills/preloaded/browser-snapshot/SKILL.md`
- [x] T010 [US1] Write the headless browser navigation and capture script in `skills/preloaded/browser-snapshot/scripts/browser_snapshot.py`
- [x] T011 [US1] Add the `browser_navigate` agent tool in `internal/agent/tools/browser_navigate.go`
- [x] T012 [US1] Register the `browser_navigate` tool in `internal/agent/tools/registry.go`
- [x] T013 [US1] Add the tool name constant in `internal/agent/tools/definitions.go`
- [x] T014 [US1] Add gateway tests for browser task routing and artifact detection in `internal/executor/browser_gateway_test.go`
- [ ] T015 [US1] Create the browser smoke test script in `scripts/browser-smoke.ps1`

**Checkpoint**: A browser navigation task from an agent tool produces a registered screenshot or page content artifact.

---

## Phase 4: User Story 2 - Keep Browser Provider Replaceable Behind The Gateway (Priority: P2)

**Goal**: Ensure the browser provider sits behind a replaceable interface seam and provider errors surface as structured failures.

**Independent Test**: A reviewer can inspect the gateway and provider seam and confirm the browser backend implements the `BrowserProvider` interface, with no product-facing contract leakage.

### Implementation for User Story 2

- [ ] T016 [US2] Add browser provider capability reporting and availability checks in `internal/executor/browser_provider.go`
- [ ] T017 [P] [US2] Add gateway tests for unknown browser provider and unavailable provider error handling in `internal/executor/browser_gateway_test.go`
- [ ] T018 [US2] Document Xelora-owned versus provider-owned boundary invariants for browser tasks in `internal/executor/browser_types.go`

**Checkpoint**: The browser provider seam is replaceable and provider errors are structured.

---

## Phase 5: User Story 3 - Integrate Browser Artifacts Into The Conversation And Workspace Flow (Priority: P3)

**Goal**: Browser artifacts inherit workspace binding, boundary enforcement, and the same preview/download path as file-producing skills.

**Independent Test**: After a browser task in a workspace-bound conversation, the screenshot artifact is registered with the correct workspace ID and respects boundary checks.

### Implementation for User Story 3

- [ ] T019 [US3] Add gateway tests for bound-workspace browser artifact routing in `internal/executor/browser_gateway_test.go`
- [ ] T020 [P] [US3] Add gateway tests for unbound conversation fallback to skill-private path in `internal/executor/browser_gateway_test.go`
- [ ] T021 [US3] Add gateway tests for boundary enforcement and path-escape rejection on browser artifacts in `internal/executor/browser_gateway_test.go`
- [ ] T022 [US3] Document the browser artifact routing and boundary contract in `specs/003-browser-automation/quickstart.md`

**Checkpoint**: Browser artifacts respect workspace binding and boundary enforcement identically to file-producing skills.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish the feature handoff cleanly and validate cross-artifact consistency.

- [ ] T023 [P] Refresh the feature summary and rollout note in `README.md`
- [ ] T024 [P] Update the runtime reference module catalog for browser automation in `docs/customizations/runtime-reference/module-catalog.yaml`
- [ ] T025 [P] Update the provider matrix for browser automation provider in `docs/customizations/runtime-reference/provider-matrix.yaml`
- [ ] T026 Perform a final consistency pass across `AGENTS.md`, `docs/customizations/TASKS.md`, and `specs/003-browser-automation/tasks.md`
- [ ] T027 Run the manual validation flow and record the final expected checks in `specs/003-browser-automation/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup**: No dependencies
- **Phase 2: Foundational**: Depends on Phase 1 and blocks all user stories
- **Phase 3: US1**: Depends on Phase 2
- **Phase 4: US2**: Depends on US1 because the provider must exist before it can be validated for replaceability
- **Phase 5: US3**: Depends on US1 and US2 because boundary enforcement validates the finalized provider and artifact contract
- **Phase 6: Polish**: Depends on all earlier phases

### User Story Dependencies

- **US1 (P1)**: Starts first and establishes the browser task dispatch, provider, and artifact flow
- **US2 (P2)**: Builds on US1 by validating the provider seam and error handling
- **US3 (P3)**: Builds on US1 and US2 by enforcing workspace binding and boundary rules on browser artifacts

### Parallel Opportunities

- `T001`, `T002`, and `T003` can proceed in parallel at setup time
- `T005` can run in parallel after `T004`
- `T009` and `T010` can run in parallel after `T008`
- `T017` can run in parallel with `T016`
- `T019`, `T020`, and `T021` can be prepared in parallel after US1 and US2
- `T023`, `T024`, and `T025` can run in parallel during polish

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. Stop and validate that an agent can navigate to a URL and produce a registered artifact

### Incremental Delivery

1. Establish browser provider primitives and gateway integration
2. Deliver the browser navigation tool and snapshot skill
3. Validate provider replaceability and error handling
4. Enforce workspace binding and boundary rules on browser artifacts
5. Finish with docs and validation cleanup

### Suggested MVP Scope

- Phase 1
- Phase 2
- Phase 3 (US1)

---

## Notes

- Keep browser provider behavior behind the `BrowserProvider` interface so the sandbox image and CDP library remain replaceable.
- Do not register browser artifacts through a parallel path; reuse the gateway file-snapshot diffing.
- Preserve compatibility for conversations that never use browser tasks.
