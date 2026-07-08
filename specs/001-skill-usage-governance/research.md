# Research: CubeSandbox-backed Web Agent Runtime

## Decision: Use CubeSandbox as the first production-oriented sandbox backend

**Rationale**: CubeSandbox is explicitly positioned as a secure sandbox service for AI agents and advertises E2B SDK compatibility. This maps well to Xelora's need for an independent execution backend that can run code and development tasks outside the main web application process.

**Alternatives considered**:

- E2B: Good API model and ecosystem, but the common path is hosted usage and self-hosting can add operational complexity.
- Daytona: Strong workspace/sandbox concepts, but current public project maintenance status makes it a weaker long-term base.
- Fully custom Docker runner: Fastest to start, but weaker isolation and higher long-term maintenance burden.

## Decision: Run CubeSandbox from WSL/Linux during local development

**Rationale**: CubeSandbox's documented setup expects Linux virtualization capabilities such as KVM/PVM paths. Running it from WSL/Linux keeps the main Xelora Docker Desktop deployment simple while allowing the sandbox layer to use the environment it expects.

**Alternatives considered**:

- Embed CubeSandbox into `docker-compose.yml`: simpler operator story on paper, but likely fights the sandbox runtime's host requirements.
- Require a remote Linux server from day one: closer to production, but slows local development and iteration.

## Decision: Keep workspace ownership in Xelora/Gateway

**Rationale**: The user selected a hybrid workspace model: Xelora/Gateway owns the session workspace and CubeSandbox mounts or maps it during execution. This protects Xelora from provider lock-in and keeps files, artifacts, permissions, and history under product control.

**Alternatives considered**:

- CubeSandbox owns complete session workspace: simpler adapter, but Xelora becomes dependent on CubeSandbox storage semantics.
- Temporary workspace per job: safer and simpler cleanup, but loses the desired Codex-like session continuity.

## Decision: Use a hybrid job model

**Rationale**: Chat sessions should behave like persistent working environments, but system probes, conversion tasks, and maintenance jobs should be able to run as one-off jobs.

**Alternatives considered**:

- Session-only jobs: easier mental model, but awkward for health checks and stateless tooling.
- Job-only model: clean isolation, but too far from the desired persistent workspace experience.

## Decision: Start with medium-restricted permissions

**Rationale**: The runtime should support normal development work such as installing dependencies, running tests, building code, editing files, and using Git, while blocking host-sensitive writes, unauthorized mounts, privilege escalation, runaway processes, and unclear external network access.

**Alternatives considered**:

- Strongly restricted: safer, but too limiting for development tasks.
- Highly permissive: faster for prototypes, but too risky for a multi-user web product.

## Decision: Add a local provider stub before the CubeSandbox adapter

**Rationale**: The product contract can be validated independently from CubeSandbox setup. The local provider is not the target production backend; it is a development bridge for API, frontend, artifact, and state-machine verification.

**Alternatives considered**:

- Build CubeSandbox adapter first: cleaner target alignment, but environment setup may block product flow validation.
- Skip local execution entirely and mock responses: too shallow; it would not validate real file creation, path checks, or artifact registration.

## Decision: Treat artifacts as first-class records

**Rationale**: The earlier web skill tests showed that a model response containing Markdown text is not enough. A successful file task must create a real file, register metadata, and expose preview/download actions in the web UI.

**Alternatives considered**:

- Parse generated files from stdout: fragile and hard to audit.
- Let each skill write arbitrary paths: flexible, but unsafe and difficult to present in the UI.
