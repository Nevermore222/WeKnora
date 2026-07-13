# Skill Studio Design

**Date:** 2026-07-13

**Status:** Draft for user review

## Goal

Make skills the primary maintainable capability unit in Xelora. Knowledge bases
remain independent evidence stores, agents remain lightweight role and routing
entry points, and detailed business logic, document workflows, templates,
scripts, validation rules, and artifact expectations live inside skills.

The first release should improve the maintenance and execution experience for
existing skills without creating a heavy business-domain platform.

## Product Principle

Xelora should be configured as:

```text
Knowledge bases = facts and evidence
Agents = role entry points and skill policy
System prompts = short routing and behavior constraints
Skills = business methods, execution scripts, templates, validation, artifacts
Artifacts = real output files and audit evidence
```

This keeps business adaptation reusable. A manager assistant, migration
assistant, or requirements assistant can share skills while using different
knowledge-base scopes and concise role prompts.

## Current Baseline

The repository already has several important foundations:

- Built-in and custom agents with configurable tools, knowledge-base scope, and
  skill selection modes.
- Preloaded skills under `skills/preloaded/`.
- Workspace-bound execution that writes real artifacts into the conversation
  workspace.
- `execute_skill_script` and `read_skill` as the primary skill invocation path.
- OfficeCLI-backed document editing through `officecli-document-editing`.
- Slash input support for selecting skill-like directives without showing long
  hidden instructions in the user input.

The gap is not raw execution capability. The gap is that administrators and
developers do not yet have a mature way to inspect, test, debug, govern, and
bind all available skills like they would manage capabilities in a modern IDE.

## First Release Scope

The first release focuses on existing skills and execution governance.

### Skill Library

Add a management surface that lists all available skills:

- Built-in/preloaded skills.
- Installed skills.
- Future tenant or business custom skills.

Each skill card should show:

- Name and short description.
- Source: built-in, installed, or custom.
- Status: enabled, disabled, invalid, or unavailable.
- Main script entry points discovered from the skill folder.
- Supported file or task hints when declared in `SKILL.md`.
- Last execution summary when available.

This page is a visibility and navigation layer. It should not require a new
business-domain model.

### Skill Detail

Each skill detail page should expose:

- Rendered `SKILL.md`.
- File tree for scripts, examples, and test fixtures.
- Declared script paths and example requests.
- Required tools or provider assumptions.
- Known output artifact types.
- Recent execution attempts and failures.

The first release may be read-only for skill source files. Editing can be added
later after execution and validation are stable.

### Skill Test Runner

Administrators and developers need a direct way to run a skill without relying
on a model to guess the call shape.

The test runner should support:

- Select a skill.
- Select or enter a script path.
- Enter `args` and JSON/stdin input.
- Choose a bound workspace for file-producing tests.
- Execute through the same gateway used by agents.
- Show validation errors before execution where possible.
- Show stdout, stderr, exit code, structured tool result, and artifacts.

Success means the skill creates or modifies expected artifacts and the artifact
registry reports them. A text-only success message is not enough.

### Agent Skill Policy

Agent configuration should keep using the existing skill selection model, but
make it easier to understand:

- `none`: the agent cannot use skills.
- `selected`: the agent can use only selected skills.
- `all`: the agent can use all enabled skills.

For business-facing agents, the recommended default is `selected`. This keeps
the agent prompt short while allowing administrators to expose only the skills
that match the role.

Later, a stricter option can be added:

- `explicit_only`: the agent may run a skill only when the user selected it
  with `/`.

This option is useful for high-risk file editing or expensive batch generation,
but it is not required for the first release.

### Slash Skill Picker

The chat input should support a Codex-like `/` picker for skills:

- Search by skill name, alias, capability, and file type.
- Insert a visible chip instead of long text.
- Hide the detailed instruction inside the directive metadata.
- Tell the agent to `read_skill` first and follow that skill's execution
  contract.

The picker should not duplicate the whole skill body into the input. It should
only select the skill and add concise routing constraints.

### Execution Trace

Every skill execution shown in the chat timeline should make failures
actionable:

- Skill name.
- Script path.
- Args and input summary.
- Workspace binding.
- Duration.
- Exit code.
- stdout and stderr snippets.
- Artifact list.
- Parameter validation failures.

This is the main defense against fragile behavior such as missing `script_path`,
malformed JSON input, or an agent claiming success after a failed script.

## Non-Goals

The first release should not include:

- A new business-domain platform.
- Full web editing of skill source code.
- Network marketplace installation UI.
- Multi-version skill publishing workflow.
- Approval workflows for enterprise skill release.
- A full batch job scheduler.
- Per-skill billing or quota management.

These are useful later, but they are not required to make skills easier to
maintain and execute.

## Data Model Direction

Prefer deriving skill metadata from the existing filesystem first:

```text
skills/preloaded/<skill>/SKILL.md
skills/preloaded/<skill>/scripts/*
skills/preloaded/<skill>/examples/*
```

Add database records only for platform-specific state:

- Enabled/disabled state.
- Source classification.
- Tenant visibility.
- Last execution summary.
- Optional aliases for slash search.
- Optional policy overrides.

This avoids duplicating `SKILL.md` as a second source of truth.

## Execution Flow

Recommended skill execution path:

1. User selects an agent and optionally selects a skill with `/`.
2. Agent prompt stays concise and instructs the model to use skills for
   concrete file or business workflow tasks.
3. Model calls `read_skill` for the selected or relevant skill.
4. Model calls `execute_skill_script` using the script path declared by the
   skill.
5. Executor gateway runs the script in the bound workspace.
6. Gateway detects artifacts and returns structured execution details.
7. Chat timeline shows the execution trace and artifact list.
8. Final answer summarizes result, artifact paths, evidence, and unresolved
   risks.

## Quality Rules For Skills

Recommended standard for every business skill:

- `SKILL.md` explains when to use the skill and when not to use it.
- `SKILL.md` declares the canonical script path for execution.
- Example requests are included for common operations.
- Scripts reject unsafe paths and invalid parameters.
- File-producing scripts validate expected output files.
- Failure messages are actionable.
- Tests or smoke examples exist for critical paths.
- Artifact expectations are documented.

## Recommended Implementation Order

1. Skill library API and page: list available skills and basic metadata.
2. Skill detail page: render `SKILL.md`, scripts, examples, and status.
3. Skill test runner: execute a selected skill through the existing gateway.
4. Execution trace polish: expose script path, args, stderr/stdout, and
   artifacts consistently.
5. Agent skill policy UI cleanup: make selected skills and modes clearer.
6. Slash picker integration: search real skills and insert hidden directives.
7. Later: skill creation, installation, versioning, and marketplace support.

## Verification

The first release is successful when:

- An administrator can see every enabled and disabled skill in one place.
- A developer can open a skill, inspect its instructions, and run a test request.
- A failed skill call shows exactly whether the issue was parameters, JSON,
  script execution, workspace binding, or artifact registration.
- A custom agent can be restricted to a small selected set of skills.
- A user can choose a skill with `/` without exposing long control text in the
  chat input.
- Office file generation and editing still go through
  `officecli-document-editing` and produce real artifacts.

## Open Design Choice

The only major sequencing choice is whether skill creation and network
installation come before or after the test runner.

Recommended decision: build the test runner first. Creation and installation
are only valuable if administrators can immediately validate that the new skill
works in the same execution path used by agents.
