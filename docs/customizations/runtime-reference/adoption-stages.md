# Xelora Runtime Adoption Stages

## Stage 1: Controlled Docker Executor Baseline

Goal: Land the first usable execution baseline without losing Xelora control of
workspaces, artifacts, policy, or execution history.

Included module families:

- sandbox-execution
- execution-gateway
- workspace-management
- artifact-management

Entry criteria:

- Controlled Docker Executor is the active local validation provider in repo context files.
- OpenSandbox is marked as experimental rather than blocking.
- Provider seam remains intact in `internal/executor/`.

Exit criteria:

- Controlled Docker Executor is the documented first usable local provider path.
- OpenSandbox remains available for provider evaluation.
- Artifact-first outcomes remain part of every execution flow.

## Stage 2: File Capability Bridge

Goal: Add real file creation and file editing paths through the shared artifact model.

Included module families:

- file-capability
- artifact-management

Focus areas:

- Markdown output
- PDF conversion
- Spreadsheet handling
- Presentation generation

## Stage 3: Browser Automation Path

Goal: Add the first browser provider path without coupling browser semantics to
the sandbox substrate.

Included module families:

- browser-automation
- artifact-management
- execution-gateway

Focus areas:

- browser task orchestration
- screenshots and page artifacts
- provider-isolated execution details

## Stage 4: Observability And Hardening

Goal: Improve runtime durability, provider routing, quota handling, and audit depth.

Included module families:

- observability-audit
- execution-gateway
- sandbox-execution

Focus areas:

- provider health visibility
- quota and routing policy
- artifact traceability
- execution audit history

## Stage 5: Stronger Sandbox Provider Evaluation

Goal: Evaluate mature sandbox providers after the first local execution path
has proven Xelora-owned workspace, job, and artifact contracts.

Included module families:

- sandbox-execution
- execution-gateway
- observability-audit

Focus areas:

- OpenSandbox retest on a compatible Linux environment
- E2B and Daytona API/workspace comparison
- gVisor hardening for Docker executor
- Kata Containers or Firecracker for production-class isolation
