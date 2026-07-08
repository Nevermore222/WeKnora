# Implementation Plan: CubeSandbox-backed Web Agent Runtime

**Branch**: `001-skill-usage-governance` | **Date**: 2026-07-08 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/001-skill-usage-governance/spec.md`, plus user decisions from the design discussion: use CubeSandbox as the sandbox base, run the sandbox environment from WSL where needed, keep Xelora as the web orchestration product, and add an independent Executor Gateway between Xelora and the sandbox backend.

## Summary

Build a governed Web Agent Runtime for Xelora where browser-side agents can execute real development tasks through an independent Executor Gateway. The first runtime target is a CubeSandbox-backed execution backend running outside the main Xelora Docker Compose stack, with session-level persistent workspaces owned by Xelora/Gateway and mounted or mapped into CubeSandbox during execution.

The plan keeps Xelora responsible for product semantics, permissions, sessions, task history, artifacts, and user experience. CubeSandbox is treated as a replaceable execution provider behind a stable Gateway contract. File generation and modification remain first-class outcomes, but the first implementation path prioritizes general development execution: script execution, dependency installation, builds, tests, Git operations, and file creation/modification inside the session workspace.

## Technical Context

**Language/Version**: Go 1.26 for the existing Xelora backend; Vue/TypeScript for the existing frontend; shell/Python/Node inside sandbox images as task runtimes.

**Primary Dependencies**: Existing Xelora app services, Docker Compose for Xelora services, WSL2/Linux host for CubeSandbox development, CubeSandbox as the sandbox provider, PostgreSQL for metadata, existing object/local storage for workspace artifacts where available.

**Storage**: PostgreSQL for workspace/job/artifact metadata; filesystem-backed workspace root for session workspaces; existing Xelora file/object storage for downloadable artifacts.

**Testing**: Go unit and integration tests for Gateway contracts and backend services; frontend component/e2e tests for job status and artifact panels; manual WSL smoke tests for CubeSandbox-backed execution.

**Target Platform**: Local Windows development with Docker Desktop for Xelora and WSL2/Linux for CubeSandbox; production should support Linux hosts capable of running CubeSandbox prerequisites.

**Project Type**: Web application plus independent execution service.

**Performance Goals**: First job status update visible in the web UI within 2 seconds after dispatch; common command jobs stream logs within 3 seconds; completed output artifacts become downloadable within 5 seconds after job completion.

**Constraints**: Default permission profile is medium-restricted: allow normal development actions in the assigned workspace, block writes outside approved paths, block host-sensitive mounts, limit long-running background processes, and make network access policy explicit per job or workspace.

**Scale/Scope**: Phase 1 supports single-node local development, one CubeSandbox provider, session-scoped persistent workspaces, synchronous or short async jobs, and web-visible logs/artifacts. Multi-node scheduling, team quotas, deep file editors, and production-grade policy UI are later phases.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

No project constitution has been initialized at `.specify/memory/constitution.md`. Until it exists, this plan applies the existing repository working rules:

- Keep changes scoped and compatible with current Xelora backend/frontend structure.
- Prefer mature open-source execution/file modules over rewriting large runtime capabilities from scratch.
- Keep web product semantics separate from sandbox provider semantics.
- Keep session workspace ownership in Xelora/Gateway so the sandbox provider remains replaceable.
- Document design and README entry points when governance or architecture changes.

**Gate Result**: PASS with note. A future `speckit-constitution` pass should formalize these principles.

## Project Structure

### Documentation (this feature)

```text
specs/001-skill-usage-governance/
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/
|   `-- executor-gateway.openapi.yaml
|-- checklists/
|   `-- requirements.md
`-- spec.md
```

### Source Code (repository root)

```text
internal/
|-- executor/
|   |-- gateway/
|   |-- workspace/
|   |-- artifact/
|   |-- policy/
|   `-- providers/
|       `-- cubesandbox/
|-- agent/
|   |-- tools/
|   `-- skills/
frontend/src/
|-- api/
|   `-- executor/
|-- views/
|   |-- chat/
|   `-- agent/
|-- components/
|   `-- executor/
docker/
|-- Dockerfile.app
`-- Dockerfile.executor-gateway
docs/customizations/
`-- EXECUTOR_RUNTIME_PLAN.md
```

**Structure Decision**: Add the execution runtime behind new `internal/executor/*` modules and frontend executor components while preserving existing `internal/agent/*` skill/tool code. The CubeSandbox adapter sits behind a provider interface so Xelora can later support another sandbox backend without rewriting session, artifact, or frontend behavior.

## Phase 0: Research Decisions

Completed in [research.md](./research.md).

Primary decisions:

- Use CubeSandbox as the first real sandbox backend, but keep it behind a Gateway provider interface.
- Run CubeSandbox from WSL/Linux for local development rather than forcing it into the existing Docker Compose stack.
- Use session-level persistent workspaces owned by Xelora/Gateway and mounted or synchronized into sandbox execution.
- Use a hybrid task model: default session jobs plus optional one-off jobs.
- Treat file outputs as artifacts registered by Gateway, not as unstructured text in chat messages.

## Phase 1: Design Artifacts

Completed in:

- [data-model.md](./data-model.md)
- [contracts/executor-gateway.openapi.yaml](./contracts/executor-gateway.openapi.yaml)
- [quickstart.md](./quickstart.md)

## Execution Roadmap

### Milestone 1: Executor Gateway Skeleton

Goal: Xelora can create a session workspace, create an execution job, and track status without yet depending on CubeSandbox.

Deliverables:

- `ExecutorWorkspace` metadata model.
- `ExecutorJob` metadata model and state machine.
- Workspace path allocator and path validation.
- API endpoints for workspace lookup, job creation, job status, logs, and artifacts.
- Minimal frontend job timeline component in the chat page.

Acceptance:

- A chat session has exactly one default workspace.
- A job can be created against that workspace and transition through queued/running/succeeded/failed.
- The frontend can show status and logs from backend state.

### Milestone 2: Local Provider Stub

Goal: Prove the full product flow before CubeSandbox integration.

Deliverables:

- Local restricted provider implementing the same provider interface.
- Command execution inside an assigned workspace only.
- Captured stdout/stderr/exit code.
- Output file discovery and artifact registration.

Acceptance:

- A web agent request can create a real file in the session workspace.
- The file appears as an artifact and can be downloaded from the web UI.
- Attempts to write outside the workspace are rejected.

### Milestone 3: CubeSandbox Provider Adapter

Goal: Replace the local provider with a CubeSandbox-backed provider while keeping Xelora/Gateway contracts unchanged.

Deliverables:

- CubeSandbox connection configuration.
- Provider adapter for sandbox creation, command execution, file sync/mount, log capture, and teardown/reuse.
- WSL development guide for running CubeSandbox locally.
- Provider health check and capability discovery.

Acceptance:

- The same web-triggered job runs through CubeSandbox.
- Workspace files persist in the Xelora/Gateway workspace after sandbox execution.
- Provider errors are surfaced as structured job errors.

### Milestone 4: Agent Tool Integration

Goal: Existing web agents can call the Executor Gateway as a normal tool path.

Deliverables:

- Replace or extend current skill execution tool behavior so executable skills can target the Gateway.
- Normalize arguments as structured arrays/objects before execution.
- Include job id, logs, exit code, and artifacts in the agent tool result.
- Add prompt guidance so agents know that success means file artifacts were created, not only text was written.

Acceptance:

- A web agent can run a development command and report generated artifacts.
- Failed commands preserve enough logs for troubleshooting.
- Existing read-skill behavior remains available.

### Milestone 5: File Capability Bridge

Goal: Make Markdown/Excel/PDF/PPT output a normal result of the same runtime.

Deliverables:

- Markdown generation and formatting flow that writes real `.md` files.
- PDF export provider path, initially via Gotenberg or a sandbox command.
- Excel/CSV read/write provider path, initially via SheetJS or a sandbox command.
- Artifact preview/download entries in the chat UI.

Acceptance:

- A user can ask an agent to create a Markdown report and download the real file.
- A user can ask an agent to modify a CSV/XLSX file and download a new version.
- Artifacts are tied to session, job, user, and workspace metadata.

### Milestone 6: Governance And Hardening

Goal: Make the runtime operationally safe enough for continued development.

Deliverables:

- Policy profiles for medium-restricted execution.
- Workspace quota and cleanup controls.
- Job timeout and process cleanup rules.
- Audit events for workspace creation, job execution, policy rejection, and artifact download.
- README/customization doc updates linking the runtime plan.

Acceptance:

- Risky operations are blocked with clear error messages.
- Operators can inspect who ran what, in which workspace, and what files were produced.
- The next backend provider can be added without changing the frontend contract.

## Provider Interface Boundary

Gateway owns:

- Session/workspace identity.
- Policy decisions.
- Artifact registry.
- Job state machine.
- User-visible logs and errors.

CubeSandbox provider owns:

- Sandbox lifecycle.
- Runtime execution.
- Runtime isolation.
- Runtime resource controls.
- Low-level command and file transfer mechanics.

The provider must not become the source of truth for business workspace identity or user-visible artifact metadata.

## Risks And Mitigations

| Risk | Mitigation |
|------|------------|
| CubeSandbox local setup is heavier than Docker Compose | Keep local provider stub and document WSL setup separately |
| Provider storage model conflicts with session workspace persistence | Keep Xelora/Gateway as workspace source of truth and use mount/sync adapters |
| Web agents continue writing text instead of real files | Change tool result contract to require artifact reporting for file tasks |
| Sandbox errors are opaque to users | Normalize provider errors into job status, stderr, and remediation hints |
| Execution freedom creates security drift | Start with medium-restricted policy and log all policy decisions |

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Add independent Executor Gateway boundary | Web agents need durable workspace execution, logs, artifacts, and provider replacement | Calling CubeSandbox directly from Xelora would bind product semantics to one provider |
| Support both session jobs and one-off jobs | Most work needs persistent chat workspaces, but maintenance and probes should remain disposable | Only session jobs would make health checks and stateless utility tasks awkward |
| Keep local provider stub before CubeSandbox adapter | CubeSandbox local setup may require WSL/Linux prerequisites | Waiting for CubeSandbox first would block frontend/backend contract validation |

## Post-Design Constitution Check

**Gate Result**: PASS with note. The plan remains aligned with the current repository constraints: scoped modules, provider isolation, mature sandbox reuse, and artifact-first verification. The only missing governance item is a formal project constitution, which should be created before broad implementation work begins.
