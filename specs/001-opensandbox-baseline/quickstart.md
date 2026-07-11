# Quickstart: Validate The Replaceable Sandbox Runtime Baseline

This quickstart is for reviewers and maintainers validating the replaceable-provider runtime plan before implementation tasks are generated.

## Validation Status

- The repo-wide planning baseline has been switched to replaceable sandbox providers.
- Controlled Docker Executor is the first usable local validation provider.
- OpenSandbox remains an experimental provider after local command-proxy smoke
  tests failed with `502`.
- The executor seam includes an OpenSandbox provider scaffold plus a legacy
  CubeSandbox compatibility path.
- `internal/executor/scripts/opensandbox_exec.py` has been syntax-checked with
  `python -m py_compile`.
- Go test execution is currently blocked in this environment because there is
  no host `go` binary available and the running `app` container does not expose
  a usable Go toolchain or repo-mounted source path for `go test`.

## 1. Read The Planning Inputs

Review:

- `specs/001-opensandbox-baseline/spec.md`
- `specs/001-opensandbox-baseline/plan.md`
- `specs/001-opensandbox-baseline/research.md`
- `docs/customizations/XELORA_FILE_CAPABILITY_PLAN.md`

Expected:

- Controlled Docker Executor is clearly identified as the current first usable local provider.
- OpenSandbox is clearly identified as experimental rather than blocking.
- The module families in scope still match Xelora's product direction.

## 2. Confirm The Sandbox Provider Decision

Review the provider assumptions captured in the planning artifacts.

Expected:

- The team agrees Controlled Docker Executor is the active local validation path.
- Reviewers understand that OpenSandbox is experimental and CubeSandbox is not the current first-baseline planning choice.
- The sandbox layer is allowed to run independently from the main Xelora web stack when that improves operability or isolation.

## 3. Confirm Ownership Boundaries

For each module family, verify:

- Xelora-owned concerns are explicit.
- Provider-owned concerns are explicit.
- Replaceable-provider boundaries are preserved.

Expected:

- No provider is allowed to take ownership of session workspace identity, artifact identity, policy decisions, or user-visible execution history.

## 4. Confirm File And Browser Module Strategy

Review the selected references:

- Browser automation: agent-browser
- Sandbox API/workspace references: E2B, Daytona
- Isolation references: gVisor, Kata Containers, Firecracker
- PDF conversion: Gotenberg
- Spreadsheet read/write: SheetJS
- Embedded office-style surfaces: Univer
- Presentation generation: PptxGenJS
- Advanced office editing: ONLYOFFICE Document Server

Expected:

- The team can explain why each reference belongs in its module family.
- The team can distinguish service-style references from embedded-SDK references.
- File artifacts remain first-class outputs even when sandbox execution is the current baseline focus.

## 5. Confirm Adoption Stages

Verify the first stage includes:

- Execution gateway
- Session workspace ownership
- Artifact-first outcomes
- Controlled Docker Executor
- Experimental OpenSandbox provider documentation

Expected:

- The first implementation wave is broad enough to be useful but narrow enough to ship.
- Later browser, file, and observability modules are clearly additive rather than mixed into the first baseline by accident.

## 6. Confirm Executor Scaffold Readiness

Review:

- `internal/executor/provider.go`
- `internal/executor/gateway.go`
- `internal/executor/opensandbox.go`
- `internal/executor/cubesandbox.go`

Expected:

- Controlled Docker Executor is the active local validation direction.
- OpenSandbox is retained only as an experimental provider scaffold.
- CubeSandbox remains only as a compatibility path.
- Xelora-owned workspace, job, policy, and artifact semantics stay outside the
  provider implementation files.
