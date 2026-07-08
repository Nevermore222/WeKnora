# Implementation Plan: Xelora Agent Runtime Reference Architecture

**Branch**: `001-agent-runtime-reference` | **Date**: 2026-07-08 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/001-agent-runtime-reference/spec.md` and the current runtime direction agreed in design discussion: use CubeSandbox as the first sandbox baseline, keep Xelora in charge of session workspaces and product semantics, and broaden the architecture to include later reference modules for browser automation, file capability services, artifact management, observability, and future provider replacement.

## Summary

Define the reference architecture that will guide Xelora's web agent runtime before implementation spreads across the codebase. The plan creates a module map, a reference-project selection record, ownership boundaries between Xelora and replaceable providers, and a staged adoption roadmap.

The output of this plan is not a full implementation yet. It is the technical baseline that tells future implementation work what must be built by Xelora, what should be delegated to mature external modules, and how the first sandbox baseline fits into the broader runtime platform.

## Technical Context

**Language/Version**: Go 1.26 for Xelora backend services; Vue/TypeScript for Xelora frontend surfaces; YAML/JSON for architecture registry artifacts.

**Primary Dependencies**: Existing Xelora backend and frontend modules; CubeSandbox as the preferred sandbox baseline; OpenHands Software Agent SDK as a reference for agent workspace and execution semantics; agent-browser as a reference for browser automation; Gotenberg, SheetJS, Univer, and PptxGenJS as file capability references.

**Storage**: Repository documentation plus future runtime metadata in PostgreSQL and filesystem/object storage remain the assumed persistence model.

**Testing**: Documentation review, schema validation for architecture registry artifacts, and future implementation conformance checks against the module boundary contract.

**Target Platform**: Xelora web product on Docker/Desktop or Linux; CubeSandbox in WSL/Linux for local development and Linux-hosted runtime for production-class execution.

**Project Type**: Web application architecture reference plus runtime module planning.

**Performance Goals**: Architecture decisions should reduce rework by enabling contributors to choose the right module and ownership boundary within one planning session; future runtime implementation should be able to derive its first usable baseline from this plan without reopening dependency scope.

**Constraints**: The architecture must favor mature external modules with minimal invasive modification, preserve Xelora ownership of session/workspace/artifact semantics, support future provider replacement, and avoid assuming that every capability must run inside the main Docker Compose stack.

**Scale/Scope**: Initial scope covers sandbox execution, execution gateway, workspace management, artifact management, browser automation, file capability services, and observability/audit integration. Deep implementation of editors, multi-node scheduling, and advanced office collaboration are outside this planning pass.

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
specs/001-agent-runtime-reference/
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/
|   `-- runtime-reference.openapi.yaml
|-- checklists/
|   `-- requirements.md
`-- spec.md
```

### Source Code (repository root)

```text
docs/customizations/
|-- XELORA_FILE_CAPABILITY_PLAN.md
|-- EXECUTOR_RUNTIME_PLAN.md
`-- runtime-reference/
    |-- module-catalog.yaml
    |-- provider-matrix.yaml
    `-- adoption-stages.md
internal/
|-- executor/
|-- agent/
|-- handler/
`-- sandbox/
frontend/src/
|-- api/
|-- components/
`-- views/
skills/preloaded/
`-- ...
```

**Structure Decision**: The reference architecture lives first as documentation and structured catalog artifacts under `docs/customizations/` and Spec Kit outputs under `specs/001-agent-runtime-reference/`. Later implementation work should map those decisions into `internal/executor/`, `internal/agent/`, `internal/sandbox/`, and relevant frontend integration points without changing the ownership boundaries recorded here.

## Phase 0: Research Decisions

Completed in [research.md](./research.md).

Primary decisions:

- Use CubeSandbox as the first sandbox execution baseline because it is purpose-built for secure, fast AI-agent sandboxes and supports single-node or multi-node scaling.
- Keep Xelora/Gateway as the owner of session workspaces, artifacts, policy decisions, and user-facing history.
- Use OpenHands Software Agent SDK as a semantic reference for workspace-oriented code agents, not as the first sandbox base.
- Use agent-browser as the primary browser automation reference.
- Use Gotenberg, SheetJS, Univer, and PptxGenJS as the initial file capability reference set, with caution that some modules are stronger for embedding while others are stronger for service-style conversion.
- Preserve a provider-agnostic contract so the first sandbox or browser provider can be swapped later.

## Phase 1: Design Artifacts

Completed in:

- [data-model.md](./data-model.md)
- [contracts/runtime-reference.openapi.yaml](./contracts/runtime-reference.openapi.yaml)
- [quickstart.md](./quickstart.md)

## Planning Roadmap

### Milestone 1: Publish The Runtime Module Catalog

Goal: Create a stable reference list of runtime module families and assign an initial preferred reference project to each family.

Deliverables:

- Canonical module families for sandbox, gateway, workspace, artifact, browser, file, and observability layers.
- Preferred reference projects and secondary alternatives.
- Ownership-boundary notes per module family.

Acceptance:

- A contributor can inspect one catalog and see what belongs to Xelora versus what belongs to an external provider.
- The first implementation planning pass no longer needs to re-debate the entire module list.

### Milestone 2: Freeze Product-Facing Contracts

Goal: Define the product-facing contracts that must survive provider replacement.

Deliverables:

- Session workspace ownership model.
- Job and artifact identity model.
- Provider capability and health model.
- Gateway contract expectations for logs, output files, policy rejections, and provider selection.

Acceptance:

- A future sandbox or browser provider can be evaluated without changing Xelora-facing concepts.
- Reviewers can reject provider-specific leakage into product semantics.

### Milestone 3: Sequence Adoption Stages

Goal: Turn the architecture into an implementation order.

Deliverables:

- Stage 1 baseline: execution gateway, session workspace model, artifact registry, local provider stub, CubeSandbox adapter.
- Stage 2 support: file capability bridge and artifact preview flows.
- Stage 3 support: browser automation provider and richer observability.
- Stage 4 hardening: quotas, provider routing, audit depth, and replacement playbooks.

Acceptance:

- The first implementation tasks can focus on the minimum useful runtime baseline.
- Later modules are positioned as additive, not scope-creep against the first baseline.

### Milestone 4: Document Replacement Strategy

Goal: Make explicit how new providers can be evaluated and adopted later.

Deliverables:

- Replacement criteria for sandbox, browser, and file providers.
- Required compatibility points for any new provider.
- Decision records for when a module is reference-only, wrapped by Gateway, or deeply integrated.

Acceptance:

- A future provider proposal can be evaluated against a known matrix instead of ad hoc judgment.
- The runtime avoids accidental lock-in to one early dependency.

## Reference Module Matrix

### Sandbox Execution

- Primary baseline: CubeSandbox
- Secondary alternatives: E2B, a local restricted provider stub
- Xelora-owned boundary: session/workspace identity, artifact identity, policy, audit, user-visible job state
- Provider-owned boundary: low-level sandbox lifecycle, process isolation, runtime resource controls

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
- Spreadsheet read/write: SheetJS
- Embedded spreadsheet/document/presentation UI: Univer
- Presentation generation: PptxGenJS
- Xelora-owned boundary: artifact identity, workflow integration, authorization, user-visible history
- Provider-owned boundary: format-specific conversion/edit mechanics

### Observability And Audit

- Primary baseline: extend Xelora's own tracing and task audit surfaces
- Secondary references: provider health/status surfaces exposed through Gateway
- Xelora-owned boundary: user-visible audit trail and cross-module correlation

## Risks And Mitigations

| Risk | Mitigation |
|------|------------|
| A mature project is attractive but poorly aligned with Xelora ownership boundaries | Require every module family to record Xelora-owned versus provider-owned concerns |
| The first sandbox choice shapes too much of the product contract | Keep CubeSandbox behind an explicit provider abstraction |
| Browser and file modules get selected independently and create conflicting user flows | Use the artifact and job model as the shared product contract |
| A reference project evolves quickly and changes APIs | Treat the project as a baseline or comparison point until integration is justified |
| Contributors still treat text responses as successful file work | Keep artifact-first outcomes as a non-negotiable runtime requirement |

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Plan covers multiple module families instead of one implementation slice | The user explicitly needs a broad reference architecture before coding deeper | Planning only CubeSandbox would not answer later module selection and ownership questions |
| Include both primary and secondary references | Future provider replacement is an explicit design goal | A single-reference plan would create avoidable lock-in pressure |
| Keep semantics and execution concerns separate | Xelora must remain the product authority | Letting providers own workspaces or artifacts would undermine later replacement |

## Post-Design Constitution Check

**Gate Result**: PASS with note. The design remains aligned with the current repository discipline: staged adoption, clear ownership boundaries, and pragmatic reuse of mature external modules. The remaining governance gap is the missing formal constitution file.
