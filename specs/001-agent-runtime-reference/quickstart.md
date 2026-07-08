# Quickstart: Validate The Runtime Reference Architecture

This quickstart is for reviewers and maintainers validating the reference architecture before implementation tasks are generated.

## 1. Read The Architecture Inputs

Review:

- `specs/001-agent-runtime-reference/spec.md`
- `specs/001-agent-runtime-reference/plan.md`
- `docs/customizations/XELORA_FILE_CAPABILITY_PLAN.md`

Expected:

- The module families in scope match the current product direction.
- The staged adoption model feels implementable.

## 2. Confirm The Sandbox Baseline Assumption

Review the CubeSandbox project and its local-development implications.

Expected:

- The team agrees CubeSandbox is the first sandbox baseline.
- Local development can tolerate WSL/Linux requirements instead of forcing sandbox execution into the main Docker Compose stack.

## 3. Confirm Ownership Boundaries

For each module family, verify:

- Xelora-owned concerns are explicit.
- Provider-owned concerns are explicit.
- Replaceable provider boundaries are preserved.

Expected:

- No provider is allowed to take ownership of session workspace identity, artifact identity, or user-visible execution history.

## 4. Confirm File And Browser Module Strategy

Review the selected references:

- Browser automation: agent-browser
- PDF conversion: Gotenberg
- Spreadsheet read/write: SheetJS
- Embedded office-style surfaces: Univer
- Presentation generation: PptxGenJS

Expected:

- The team can explain why each reference belongs in its module family.
- The team can distinguish service-style references from embedded-SDK references.

## 5. Confirm Adoption Stages

Verify the first stage includes:

- Execution gateway
- Session workspace ownership
- Artifact-first outcomes
- Local provider validation
- CubeSandbox adapter

Expected:

- The first implementation wave is broad enough to be useful but narrow enough to ship.
