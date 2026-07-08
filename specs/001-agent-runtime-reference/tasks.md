# Tasks: Xelora Agent Runtime Reference Architecture

**Input**: Design documents from `F:\Docker\WeKnora\Xelora\specs\001-agent-runtime-reference\`

**Prerequisites**: `plan.md` (required), `spec.md` (required), `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: No dedicated automated test tasks were requested in the feature specification. Validation is handled through documentation consistency checks and quickstart review.

**Organization**: Tasks are grouped by user story so each planning outcome can be completed and reviewed independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel when the task touches different files and does not depend on incomplete work
- **[Story]**: User story label for traceability (`[US1]`, `[US2]`, `[US3]`)
- Every task includes the exact file path to update

## Phase 1: Setup (Shared Documentation Entry Points)

**Purpose**: Create the stable landing zone that future agents and contributors can follow.

- [ ] T001 Create the runtime reference document index in `docs/customizations/runtime-reference/README.md`
- [ ] T002 [P] Add runtime reference document links to `README.md`
- [ ] T003 [P] Add runtime reference document links and usage notes to `docs/customizations/README-dev.md`
- [ ] T004 Sync runtime reference progress wording between `AGENTS.md` and `docs/customizations/TASKS.md`

---

## Phase 2: Foundational (Blocking Planning Artifacts)

**Purpose**: Establish the structured artifacts that all user-story outcomes depend on.

**Critical**: User-story work should not start until these files exist with stable structure.

- [ ] T005 Create the module catalog skeleton in `docs/customizations/runtime-reference/module-catalog.yaml`
- [ ] T006 [P] Create the provider matrix skeleton in `docs/customizations/runtime-reference/provider-matrix.yaml`
- [ ] T007 [P] Create the staged rollout skeleton in `docs/customizations/runtime-reference/adoption-stages.md`
- [ ] T008 Align gateway and provider vocabulary in `specs/001-agent-runtime-reference/contracts/runtime-reference.openapi.yaml`
- [ ] T009 Document artifact expectations and file ownership rules in `docs/customizations/runtime-reference/README.md`

**Checkpoint**: The runtime reference directory and contract vocabulary are ready for story-specific content.

---

## Phase 3: User Story 1 - Choose A Stable Runtime Baseline (Priority: P1)

**Goal**: Publish one clear runtime module map with preferred references and fallback candidates.

**Independent Test**: A contributor can identify the required module families and preferred open-source references by reading `module-catalog.yaml`, `provider-matrix.yaml`, and the runtime reference index.

### Implementation for User Story 1

- [ ] T010 [US1] Populate in-scope runtime module families in `docs/customizations/runtime-reference/module-catalog.yaml`
- [ ] T011 [P] [US1] Add primary baseline and secondary alternatives per module family in `docs/customizations/runtime-reference/provider-matrix.yaml`
- [ ] T012 [US1] Document the baseline lookup guide for sandbox, gateway, workspace, artifact, browser, file, and observability layers in `docs/customizations/runtime-reference/README.md`
- [ ] T013 [US1] Reconcile module-family descriptions with `specs/001-agent-runtime-reference/spec.md` and update `docs/customizations/runtime-reference/module-catalog.yaml`

**Checkpoint**: The preferred runtime baseline is discoverable without reading implementation code.

---

## Phase 4: User Story 2 - Protect Product Ownership While Reusing Mature Modules (Priority: P2)

**Goal**: Make Xelora-owned semantics explicit so provider choices cannot erode workspace, artifact, and policy control.

**Independent Test**: A reviewer can distinguish Xelora-owned concerns from provider-owned concerns for every in-scope module family using the runtime reference artifacts alone.

### Implementation for User Story 2

- [ ] T014 [US2] Add Xelora-owned and provider-owned concern lists for each module family in `docs/customizations/runtime-reference/provider-matrix.yaml`
- [ ] T015 [P] [US2] Document contract invariants and provider replacement rules in `docs/customizations/runtime-reference/README.md`
- [ ] T016 [US2] Update stable product-facing entities and provider metadata in `specs/001-agent-runtime-reference/contracts/runtime-reference.openapi.yaml`
- [ ] T017 [US2] Add the ownership-boundary review checklist to `docs/customizations/runtime-reference/README.md`

**Checkpoint**: The architecture now protects product semantics even when providers change.

---

## Phase 5: User Story 3 - Sequence The Runtime Roadmap (Priority: P3)

**Goal**: Turn the architecture into a staged implementation roadmap that the repo can execute incrementally.

**Independent Test**: A contributor can derive the first implementation slice and the follow-up slices from the staged roadmap without reopening scope.

### Implementation for User Story 3

- [ ] T018 [US3] Write Stage 1 through Stage 4 adoption steps in `docs/customizations/runtime-reference/adoption-stages.md`
- [ ] T019 [P] [US3] Map shared board items `T-007` through `T-013` into the staged roadmap in `docs/customizations/runtime-reference/adoption-stages.md`
- [ ] T020 [US3] Document the WSL/Linux local-development path and out-of-compose deployment note for CubeSandbox in `docs/customizations/runtime-reference/adoption-stages.md`
- [ ] T021 [US3] Add implementation handoff targets for `internal/executor/`, `internal/agent/`, and `internal/sandbox/` in `docs/customizations/runtime-reference/README.md`

**Checkpoint**: The roadmap shows what to build first, what follows later, and where the first code changes belong.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Keep the planning chain consistent across repo entry points and shared progress files.

- [ ] T022 [P] Validate runtime-reference usage steps in `specs/001-agent-runtime-reference/quickstart.md`
- [ ] T023 [P] Refresh runtime reference completion state in `docs/customizations/TASKS.md`
- [ ] T024 Perform a final consistency pass across `AGENTS.md`, `README.md`, `specs/001-agent-runtime-reference/tasks.md`, and `docs/customizations/runtime-reference/README.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup**: No dependencies
- **Phase 2: Foundational**: Depends on Phase 1 and blocks all user stories
- **Phase 3: US1**: Depends on Phase 2
- **Phase 4: US2**: Depends on Phase 3 outputs for module-family coverage
- **Phase 5: US3**: Depends on Phase 3 and Phase 4 so the roadmap reflects both reference selection and ownership boundaries
- **Phase 6: Polish**: Depends on all prior phases

### User Story Dependencies

- **US1 (P1)**: Starts first and defines the baseline module set
- **US2 (P2)**: Builds on US1 by locking ownership boundaries around the selected modules
- **US3 (P3)**: Builds on US1 and US2 to stage the roadmap and implementation handoff

### Parallel Opportunities

- `T002` and `T003` can run in parallel after `T001`
- `T005`, `T006`, and `T007` can be prepared in parallel once the directory entry point exists
- `T011` can run in parallel with `T012` after `T010`
- `T015` can run in parallel with `T016` after `T014`
- `T019` and `T020` can run in parallel after `T018`
- `T022` and `T023` can run in parallel during polish

---

## Parallel Example: User Story 1

```text
Task: "Add primary baseline and secondary alternatives per module family in docs/customizations/runtime-reference/provider-matrix.yaml"
Task: "Document the baseline lookup guide for sandbox, gateway, workspace, artifact, browser, file, and observability layers in docs/customizations/runtime-reference/README.md"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 and Phase 2
2. Complete Phase 3 (US1)
3. Stop and validate that contributors can identify module families and preferred references quickly

### Incremental Delivery

1. Publish the runtime reference directory and structured artifacts
2. Lock the stable baseline module map
3. Add ownership-boundary rules
4. Add the staged adoption roadmap
5. Refresh shared repo entry points and progress files

### Suggested MVP Scope

- Phase 1
- Phase 2
- Phase 3 (US1)

---

## Notes

- Keep tasks tied to exact files so future implementation work stays surgical.
- Do not treat text-only agent output as completion when a task requires a real repository artifact.
- Reuse the shared board in `docs/customizations/TASKS.md` instead of creating a parallel planning tracker.
