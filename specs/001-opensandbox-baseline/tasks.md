# Tasks: Xelora OpenSandbox Runtime Baseline

**Input**: Design documents from `F:\Docker\WeKnora\Xelora\specs\001-opensandbox-baseline\`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/runtime-baseline.openapi.yaml`, `quickstart.md`

**Tests**: The feature specification did not require a TDD-first workflow. Validation is handled through document consistency, focused Go tests around the executor seam, and OpenSandbox-oriented configuration review.

**Organization**: Tasks are grouped by user story so the planning rebase, ownership-boundary hardening, and provider-replacement preparation can each be completed and reviewed independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel when the task touches different files and does not depend on incomplete work
- **[Story]**: User story label for traceability (`[US1]`, `[US2]`, `[US3]`)
- Every task includes the exact file path to update

## Phase 1: Setup (Shared Context Rebase)

**Purpose**: Point the repo's shared context at the new OpenSandbox planning baseline before more implementation work continues.

- [x] T001 Update runtime-planning references from `specs/001-agent-runtime-reference/` to `specs/001-opensandbox-baseline/` in `AGENTS.md`
- [x] T002 [P] Add the OpenSandbox baseline spec and plan links to `docs/customizations/README-dev.md`
- [x] T003 [P] Add the OpenSandbox baseline planning chain to `docs/customizations/runtime-reference/README.md`
- [x] T004 Align the shared progress board wording with the OpenSandbox direction in `docs/customizations/TASKS.md`

---

## Phase 2: Foundational (Blocking Planning And Provider Prerequisites)

**Purpose**: Create the shared artifacts and provider vocabulary that block all later story work.

**Critical**: No user-story work should begin until these tasks are complete.

- [x] T005 Create the OpenSandbox-first module catalog in `docs/customizations/runtime-reference/module-catalog.yaml`
- [x] T006 [P] Create the provider comparison matrix in `docs/customizations/runtime-reference/provider-matrix.yaml`
- [x] T007 [P] Create the staged adoption roadmap in `docs/customizations/runtime-reference/adoption-stages.md`
- [x] T008 Replace CubeSandbox-first environment guidance with OpenSandbox-first guidance in `.env.example`
- [x] T009 Replace CubeSandbox-first skill-execution environment guidance with OpenSandbox-first guidance in `docs/agent-skills.md`
- [x] T010 Document the OpenSandbox provider slot and config expectations in `internal/executor/provider.go`

**Checkpoint**: Shared repo context, config vocabulary, and provider reference artifacts are aligned on OpenSandbox.

---

## Phase 3: User Story 1 - Rebase The Runtime Plan On OpenSandbox (Priority: P1) MVP

**Goal**: Make Controlled Docker Executor the explicit first usable local provider across the active planning and execution entry points, with OpenSandbox retained as experimental.

**Independent Test**: A contributor can read the repo context files and identify Controlled Docker Executor as the first local validation path without consulting older CubeSandbox- or OpenSandbox-first documents.

### Implementation for User Story 1

- [x] T011 [US1] Update the runtime baseline section from CubeSandbox to OpenSandbox in `AGENTS.md`
- [x] T012 [P] [US1] Reword pending runtime work items `T-009` and `T-010` around OpenSandbox in `docs/customizations/TASKS.md`
- [x] T013 [P] [US1] Publish the OpenSandbox-first reference summary in `docs/customizations/runtime-reference/README.md`
- [x] T014 [US1] Populate the active sandbox provider direction in `docs/customizations/runtime-reference/module-catalog.yaml`
- [x] T015 [US1] Populate OpenSandbox, CubeSandbox, E2B, and local-stub roles in `docs/customizations/runtime-reference/provider-matrix.yaml`
- [x] T016 [US1] Add the OpenSandbox-first Stage 1 baseline narrative in `docs/customizations/runtime-reference/adoption-stages.md`

**Checkpoint**: The active planning baseline is now visibly OpenSandbox across repo context and runtime-reference artifacts.

---

## Phase 4: User Story 2 - Preserve Xelora-Owned Product Semantics (Priority: P2)

**Goal**: Lock the ownership boundary so adopting OpenSandbox does not weaken Xelora control of workspaces, artifacts, policy, or user-visible execution history.

**Independent Test**: A reviewer can inspect the plan artifacts and executor seam files and see a clear split between Xelora-owned contracts and provider-owned execution mechanics.

### Implementation for User Story 2

- [x] T017 [US2] Record Xelora-owned versus provider-owned concerns for each module family in `docs/customizations/runtime-reference/provider-matrix.yaml`
- [x] T018 [P] [US2] Add contract invariants and replacement rules to `docs/customizations/runtime-reference/README.md`
- [x] T019 [P] [US2] Align the OpenAPI contract entities with OpenSandbox-era ownership language in `specs/001-opensandbox-baseline/contracts/runtime-baseline.openapi.yaml`
- [x] T020 [US2] Add OpenSandbox-oriented provider metadata and invariant comments in `internal/executor/types.go`
- [x] T021 [US2] Document Xelora-owned gateway responsibilities and provider boundaries in `internal/executor/gateway.go`
- [x] T022 [US2] Update executor seam tests to reflect provider-agnostic ownership expectations in `internal/executor/gateway_test.go`

**Checkpoint**: Xelora-owned semantics are explicit in both design artifacts and the executor seam.

---

## Phase 5: User Story 3 - Keep The Runtime Extensible Beyond One Sandbox (Priority: P3)

**Goal**: Prepare the repo for OpenSandbox integration without locking future browser, file, or alternate-provider work to one implementation path.

**Independent Test**: A contributor can follow the roadmap and provider seam to add OpenSandbox now while still seeing a clean path for local stub, browser, file, and future provider replacement work.

### Implementation for User Story 3

- [x] T023 [US3] Rename the pending sandbox integration path from CubeSandbox to OpenSandbox in `docs/customizations/TASKS.md`
- [x] T024 [P] [US3] Add Stage 2 through Stage 4 follow-up module sequencing in `docs/customizations/runtime-reference/adoption-stages.md`
- [x] T025 [P] [US3] Introduce an OpenSandbox provider implementation scaffold in `internal/executor/opensandbox.go`
- [x] T026 [US3] Move CubeSandbox-specific helper logic behind a deprecated compatibility path in `internal/executor/cubesandbox.go`
- [x] T027 [US3] Add an OpenSandbox helper execution script scaffold in `internal/executor/scripts/opensandbox_exec.py`
- [x] T028 [US3] Update skill manager execution notes for provider replacement and artifact-first outcomes in `internal/agent/skills/manager.go`
- [x] T029 [US3] Extend app-container dependency guidance for the OpenSandbox helper path in `docker/Dockerfile.app`

**Checkpoint**: The repo has a forward path for OpenSandbox integration while keeping the executor seam replaceable.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish the planning handoff cleanly and remove ambiguity from parallel agent work.

- [x] T030 [P] Validate the OpenSandbox planning flow and review steps in `specs/001-opensandbox-baseline/quickstart.md`
- [x] T031 [P] Add an OpenSandbox planning completion note to `specs/001-opensandbox-baseline/checklists/requirements.md`
- [x] T032 Refresh the runtime reference README with links to `spec.md`, `plan.md`, `research.md`, and `tasks.md` in `docs/customizations/runtime-reference/README.md`
- [x] T033 Perform a final consistency pass across `AGENTS.md`, `docs/customizations/TASKS.md`, `docs/agent-skills.md`, and `specs/001-opensandbox-baseline/tasks.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup**: No dependencies
- **Phase 2: Foundational**: Depends on Phase 1 and blocks all user stories
- **Phase 3: US1**: Depends on Phase 2
- **Phase 4: US2**: Depends on US1 outputs because ownership boundaries build on the chosen baseline
- **Phase 5: US3**: Depends on US1 and US2 so the implementation seam reflects the finalized baseline and ownership rules
- **Phase 6: Polish**: Depends on all earlier phases

### User Story Dependencies

- **US1 (P1)**: Starts first and establishes OpenSandbox as the active baseline
- **US2 (P2)**: Builds on US1 by fixing Xelora-owned versus provider-owned boundaries
- **US3 (P3)**: Builds on US1 and US2 to keep the executor seam and roadmap replaceable

### Parallel Opportunities

- `T002` and `T003` can run in parallel after `T001`
- `T005`, `T006`, and `T007` can run in parallel after the setup phase
- `T008` and `T009` can run in parallel once the provider vocabulary is decided
- `T012` and `T013` can run in parallel after `T011`
- `T018` and `T019` can run in parallel after `T017`
- `T024` and `T025` can run in parallel after `T023`
- `T030` and `T031` can run in parallel in the polish phase

---

## Parallel Example: User Story 1

```text
Task: "Reword pending runtime work items T-009 and T-010 around OpenSandbox in docs/customizations/TASKS.md"
Task: "Publish the OpenSandbox-first reference summary in docs/customizations/runtime-reference/README.md"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. Stop and validate that all active repo entry points now identify Controlled Docker Executor as the first usable local provider and OpenSandbox as experimental

### Incremental Delivery

1. Repoint shared context and progress files to the new OpenSandbox feature
2. Freeze the OpenSandbox baseline across runtime-reference artifacts
3. Lock Xelora-owned contracts around that baseline
4. Introduce the OpenSandbox implementation seam and helper scaffold
5. Finish with quickstart and consistency validation

### Suggested MVP Scope

- Phase 1
- Phase 2
- Phase 3 (US1)

---

## Notes

- Keep tasks tied to exact files so later execution stays surgical.
- Do not keep CubeSandbox as the active wording in shared planning entry points once US1 is complete.
- Preserve the provider seam instead of hard-coding OpenSandbox behavior directly into product-facing modules.
- Treat file artifacts and execution history as first-class outcomes during all later implementation work.
