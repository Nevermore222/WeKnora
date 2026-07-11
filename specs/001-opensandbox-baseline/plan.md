# Implementation Plan: Xelora Replaceable Sandbox Runtime Baseline

**Branch**: `001-opensandbox-baseline` | **Date**: 2026-07-09 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/001-opensandbox-baseline/spec.md`

## Summary

Rebase Xelora's runtime architecture plan on replaceable sandbox providers while preserving Xelora ownership of session workspaces, artifact records, policy decisions, and user-visible history. The plan keeps independent sandbox execution as the target architecture, uses a controlled Docker executor as the first usable local provider, and retains OpenSandbox as an experimental provider after local command-proxy validation failed.

## Technical Context

**Language/Version**: Go 1.26 for backend services; Vue 3.5 and TypeScript 6 for frontend surfaces; YAML and Markdown for planning artifacts

**Primary Dependencies**: Existing Xelora backend and frontend modules; Controlled Docker Executor as the first usable local provider; OpenSandbox as an experimental sandbox provider; OpenHands Software Agent SDK as semantic workspace reference; E2B and Daytona as sandbox API/workspace references; agent-browser as browser automation reference; Gotenberg, SheetJS, Univer, PptxGenJS, and ONLYOFFICE as file capability references

**Storage**: Repository documentation for planning artifacts; PostgreSQL and filesystem or object storage remain the assumed runtime persistence model for future implementation

**Testing**: Documentation review, contract review, future Go and frontend test suites, and implementation conformance checks against ownership boundaries

**Target Platform**: Xelora web product on Docker Desktop or Linux for main app services, with execution providers allowed to run as independently managed services when that improves operability or isolation

**Project Type**: Web application architecture reference plus runtime module planning

**Performance Goals**: Provide a first usable baseline that supports real task execution and file-artifact workflows without blocking on one external provider

**Constraints**: Reuse mature open-source modules with minimal invasive modification; preserve Xelora-owned product semantics; keep provider replacement viable; avoid coupling browser and file modules to one sandbox-specific workflow

**Scale/Scope**: Initial scope covers sandbox execution, execution gateway, workspace management, artifact management, browser automation, file capability services, and observability or audit integration; deep implementation of collaborative editors, cluster scheduling policy, and advanced office workflows stays out of this planning pass

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

No project constitution has been initialized at `.specify/memory/constitution.md`. Until it exists, this plan uses the current repository operating principles:

- Prefer mature open-source modules over large custom rewrites.
- Keep product ownership boundaries explicit.
- Preserve replaceability of low-level runtime providers.
- Keep documentation discoverable from the repository.
- Stage adoption so the first runtime baseline remains buildable and testable.

**Gate Result**: PASS with note. A later constitution pass should formalize these rules before broader implementation begins.

## Project Structure

### Documentation (this feature)

```text
specs/001-opensandbox-baseline/
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/
|   `-- runtime-baseline.openapi.yaml
`-- spec.md
```

### Source Code (repository root)

```text
docs/customizations/
|-- TASKS.md
|-- XELORA_FILE_CAPABILITY_PLAN.md
`-- runtime-reference/
    |-- module-catalog.yaml
    |-- provider-matrix.yaml
    `-- adoption-stages.md
internal/
|-- agent/
|-- executor/
|-- handler/
|-- runtime/
`-- sandbox/
frontend/src/
|-- api/
|-- components/
|-- stores/
`-- views/
skills/preloaded/
`-- ...
```

**Structure Decision**: The replaceable-provider baseline is planned first as documentation and structured contract artifacts under `specs/001-opensandbox-baseline/` and `docs/customizations/`. Later implementation work should map those decisions into `internal/executor/`, `internal/agent/`, `internal/sandbox/`, and relevant frontend integration points without moving ownership boundaries into provider-specific code.

## Phase 0: Research Decisions

Completed in [research.md](./research.md).

Primary decisions:

- Use a controlled Docker executor as the first usable local provider.
- Keep OpenSandbox as an experimental provider and mature reference after local command-proxy smoke tests failed.
- Keep Xelora and its future execution gateway as the owner of session workspaces, artifacts, policy decisions, and user-facing execution history.
- Treat execution providers as independently replaceable services rather than forcing product semantics into provider-specific code.
- Keep OpenHands Software Agent SDK as the semantic reference for workspace-oriented agent behavior.
- Track E2B and Daytona as references for sandbox API and developer-workspace product shape.
- Preserve the existing browser and file capability reference split so those modules stay replaceable and artifact-first.
- Keep a staged adoption model so the first implementation wave remains buildable.

## Phase 1: Design Artifacts

Completed in:

- [data-model.md](./data-model.md)
- [contracts/runtime-baseline.openapi.yaml](./contracts/runtime-baseline.openapi.yaml)
- [quickstart.md](./quickstart.md)

## Planning Roadmap

### Milestone 1: Freeze The Replaceable Provider Baseline

Goal: Publish a stable runtime reference that makes Controlled Docker Executor the first usable local provider, keeps OpenSandbox experimental, and removes ambiguity from future planning work.

Deliverables:

- Updated sandbox provider baseline record
- Runtime module families and reference-project mapping
- Ownership-boundary notes per module family

Acceptance:

- A contributor can inspect one planning set and see that Controlled Docker Executor is the active local validation path.
- Reviewers understand that OpenSandbox remains available for evaluation but does not block first usable runtime work.

### Milestone 2: Freeze Xelora-Owned Contracts

Goal: Define the product-facing contracts that must survive provider replacement.

Deliverables:

- Session workspace ownership model
- Job and artifact identity model
- Provider capability and health expectations
- Gateway contract expectations for logs, output files, policy rejections, and provider selection

Acceptance:

- A future sandbox or browser provider can be evaluated without changing Xelora-facing concepts.
- Reviewers can reject provider-specific leakage into product semantics.

### Milestone 3: Sequence Adoption Stages

Goal: Turn the reference architecture into an implementation order anchored on a controlled local provider and replaceable sandbox adapters.

Deliverables:

- Stage 1 baseline: execution gateway, session workspace model, artifact registry, controlled Docker executor, OpenSandbox experimental adapter documentation
- Stage 2 support: file capability bridge and artifact preview flows
- Stage 3 support: browser automation provider and richer observability
- Stage 4 hardening: quotas, provider routing, audit depth, and replacement playbooks

Acceptance:

- The first implementation tasks can focus on the minimum useful runtime baseline.
- Later modules are positioned as additive, not as scope creep in the first execution wave.

### Milestone 4: Document Replacement Strategy

Goal: Make explicit how later providers can be evaluated and adopted after the first local provider.

Deliverables:

- Replacement criteria for sandbox, browser, and file providers
- Required compatibility points for any new provider
- Decision records for when a module is reference-only, wrapped by gateway, or deeply integrated

Acceptance:

- A future provider proposal can be evaluated against a known matrix instead of ad hoc judgment.
- The runtime avoids accidental lock-in to one early dependency.

## Reference Module Matrix

### Sandbox Execution

- Primary local baseline: Controlled Docker Executor
- Experimental provider: OpenSandbox
- API/workspace references: E2B, Daytona
- Stronger isolation references: gVisor, Kata Containers, Firecracker
- Compatibility alternative: CubeSandbox
- Xelora-owned boundary: session or workspace identity, artifact identity, policy, audit, user-visible job state
- Provider-owned boundary: low-level sandbox lifecycle, process isolation, runtime resource controls, execution surface mechanics

### Execution Semantics

- Primary semantic reference: OpenHands Software Agent SDK
- Secondary alternatives: other agent runtime SDKs when they preserve workspace semantics
- Xelora-owned boundary: product workflow, tool invocation path, session continuity
- Reference use: conceptual and contract guidance rather than first runtime backend

### Browser Automation

- Primary baseline: agent-browser
- Secondary alternatives: later MCP- or CDP-based providers
- Xelora-owned boundary: task intent, session linkage, artifact and log handling
- Provider-owned boundary: browser launch, DOM interaction, screenshots, page automation mechanics

### File Capability Services

- PDF conversion: Gotenberg
- Spreadsheet read or write: SheetJS
- Embedded spreadsheet, document, or presentation UI: Univer
- Presentation generation: PptxGenJS
- Xelora-owned boundary: artifact identity, workflow integration, authorization, user-visible history
- Provider-owned boundary: format-specific conversion and editing mechanics

### Observability And Audit

- Primary baseline: extend Xelora's own tracing and task-audit surfaces
- Secondary references: provider health and status surfaces exposed through the execution gateway
- Xelora-owned boundary: user-visible audit trail and cross-module correlation

## Risks And Mitigations

| Risk | Mitigation |
|------|------------|
| Older planning artifacts still bias contributors toward CubeSandbox or OpenSandbox as a single baseline | Make Controlled Docker Executor the explicit first usable local provider and OpenSandbox an experimental provider in active entry points |
| The sandbox baseline starts to absorb file, browser, or artifact semantics | Keep those concerns behind shared Xelora-owned contracts and separate module families |
| A provider becomes attractive enough that provider boundaries are weakened | Require every module family to record Xelora-owned versus provider-owned concerns |
| The first implementation wave becomes too broad again | Keep Stage 1 limited to gateway, workspace, artifact registry, controlled Docker executor, and experimental provider documentation |
| Future provider replacement becomes expensive because early contracts leak provider details | Preserve provider-agnostic invariants and publish them before implementation expands |

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Plan covers multiple runtime module families instead of just the sandbox provider | The user needs a durable runtime roadmap, not a one-off provider swap | Planning only one provider adapter would not prepare browser, file, and artifact work |
| Include both primary and secondary references | Future provider replacement is an explicit design goal | A single-reference plan would create avoidable lock-in pressure |
| Keep execution semantics and execution substrate as separate concerns | Xelora must remain the product authority | Letting the provider define workspaces or user-visible history would undermine later replacement |
