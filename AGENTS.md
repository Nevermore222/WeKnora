# Xelora Agent Context

This repository uses `AGENTS.md` as the default coding-agent context file.
All future agents working in this repo should read this file first and follow
the linked planning and progress documents before making substantial changes.

## Required Reading

Before planning, coding, or reviewing runtime-related work, read these files in order:

1. `specs/001-opensandbox-baseline/spec.md`
2. `specs/001-opensandbox-baseline/plan.md`
3. `docs/customizations/TASKS.md`

Use `specs/001-opensandbox-baseline/spec.md` as the architecture scope,
`specs/001-opensandbox-baseline/plan.md` as the implementation baseline,
and `docs/customizations/TASKS.md` as the shared progress board.

For the browser automation feature (T-012, Stage 3), also read:
`specs/003-browser-automation/spec.md` and `specs/003-browser-automation/plan.md`.

## Working Rules

- Keep Xelora in control of session workspace identity, job identity, artifact
  identity, policy decisions, and user-visible execution history.
- Treat external runtime modules as replaceable providers unless the active
  plan explicitly says otherwise.
- Use `Controlled Docker Executor` as the first usable local provider for
  runtime validation. Keep `OpenSandbox` as an experimental provider and mature
  reference, but do not let provider-specific behavior leak into product-facing
  contracts.
- Prefer staged delivery: first baseline runtime, then file capability bridge,
  then browser automation and hardening.
- When implementation changes affect progress, update `docs/customizations/TASKS.md`
  in the same workstream so later agents inherit the latest state.
- Do not treat text-only agent responses as successful file work when the task
  is supposed to create or modify real artifacts.

## Progress Tracking

- Use `docs/customizations/TASKS.md` as the shared cross-machine progress file.
- Add new work items there before or during implementation when the task
  changes project state in a meaningful way.
- Keep task wording aligned with the current runtime architecture plan.

## Runtime Baseline

- Product shell: Xelora web app and existing agent/session model
- First usable local provider: Controlled Docker Executor
- Experimental sandbox provider: OpenSandbox
- Workspace ownership: Xelora / Executor Gateway
- Browser automation reference: agent-browser
- File capability references: Gotenberg, SheetJS, Univer, PptxGenJS

<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan
at F:\Docker\WeKnora\Xelora\specs\002-session-workspace-binding\plan.md
<!-- SPECKIT END -->
<!-- SPECKIT END -->
For the current active feature, read the plan at
`F:\Docker\WeKnora\Xelora\specs\003-browser-automation\plan.md`.
