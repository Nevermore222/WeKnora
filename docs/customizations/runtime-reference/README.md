# Xelora Runtime Reference

This directory is the landing zone for the runtime-reference artifacts defined in
`specs/001-agent-runtime-reference/`.

It is intended to hold the durable, repo-visible outputs that future agents and
contributors can inspect before changing the runtime stack.

## Planned Artifacts

- `module-catalog.yaml`: canonical runtime module families and their stage tags
- `provider-matrix.yaml`: preferred references, alternatives, ownership split,
  and replacement notes
- `adoption-stages.md`: staged rollout order from baseline runtime to later
  browser and file capabilities

## Source Of Truth

Read these files first when working on runtime execution architecture:

1. `specs/001-agent-runtime-reference/spec.md`
2. `specs/001-agent-runtime-reference/plan.md`
3. `specs/001-agent-runtime-reference/tasks.md`
4. `docs/customizations/TASKS.md`

## Working Rules

- Keep Xelora in charge of session workspace identity, job identity, artifact
  identity, policy, and user-visible execution history.
- Treat CubeSandbox as the first sandbox baseline, not as the owner of product
  semantics.
- Prefer artifact-producing capability paths for Markdown, spreadsheet, PDF,
  and presentation workflows.
- Keep provider-specific details behind replaceable contracts.
