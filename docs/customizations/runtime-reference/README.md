# Xelora Runtime Reference

This directory is the landing zone for the durable runtime-reference artifacts
defined in `specs/001-opensandbox-baseline/`.

It exists so future agents and contributors can confirm the active runtime
baseline, ownership boundaries, and rollout order before changing the executor,
artifact, browser, or file-capability stack.

## Planned Artifacts

- `module-catalog.yaml`: canonical runtime module families, stage tags, and
  primary references
- `provider-matrix.yaml`: provider roles, alternatives, ownership split, and
  replacement notes
- `adoption-stages.md`: staged rollout order from Controlled Docker Executor
  validation to later browser, file, and stronger sandbox capabilities

## Source Of Truth

Read these files first when working on runtime execution architecture:

1. `specs/001-opensandbox-baseline/spec.md`
2. `specs/001-opensandbox-baseline/plan.md`
3. `specs/001-opensandbox-baseline/tasks.md`
4. `docs/customizations/TASKS.md`

## Working Rules

- Keep Xelora in charge of session workspace identity, job identity, artifact
  identity, policy decisions, and user-visible execution history.
- Treat Controlled Docker Executor as the first usable local provider.
- Treat OpenSandbox as an experimental provider and mature reference, not as
  the owner of product semantics.
- Keep browser automation, file creation, file editing, and observability
  behind replaceable contracts instead of coupling them to one sandbox.
- Prefer artifact-producing capability paths for Markdown, spreadsheet, PDF,
  and presentation workflows.

## Contract Invariants

These rules must remain true when any execution provider is replaced:

- Workspace identity is issued and tracked by Xelora, not by the provider.
- Job state, policy checks, and execution history remain gateway-owned.
- File outputs become Xelora artifacts with stable metadata and traceability.
- Browser and file capabilities stay modular instead of being absorbed into a
  sandbox-specific workflow.

## Replacement Rules

- New providers must fit behind the executor provider seam.
- Replacing a provider must not change artifact semantics or user-visible job
  status behavior.
- Compatibility or experimental paths such as OpenSandbox and CubeSandbox can
  remain for migration, comparison, or future testing, but they are no longer
  the active local validation direction.

## Current Planning Chain

- Specification: `specs/001-opensandbox-baseline/spec.md`
- Implementation plan: `specs/001-opensandbox-baseline/plan.md`
- Research decisions: `specs/001-opensandbox-baseline/research.md`
- Task breakdown: `specs/001-opensandbox-baseline/tasks.md`
- Workspace and Office progress: `docs/customizations/WORKSPACE_OFFICE_PROGRESS.md`
- Controlled Docker smoke test: `scripts/controlled-docker-smoke.ps1`
- OfficeCLI file capability smoke test: `scripts/officecli-smoke.ps1`
- Smoke test: `docs/customizations/runtime-reference/opensandbox-smoke.md`
