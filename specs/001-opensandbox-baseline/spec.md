# Feature Specification: Xelora Replaceable Sandbox Runtime Baseline

**Feature Branch**: `001-opensandbox-baseline`

**Created**: 2026-07-09

**Status**: Draft

**Input**: User description: "Update the Xelora runtime development plan so independent sandbox execution remains the target architecture, but OpenSandbox is treated as an experimental provider after local integration issues; identify mature reference projects and prioritize a controlled Docker execution service plus independent file capability modules."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Rebase The Runtime Plan On Replaceable Providers (Priority: P1)

As the owner of Xelora secondary development, I want the runtime development plan to keep independent sandbox execution as the target architecture while avoiding lock-in to one sandbox implementation, so that the team can continue toward real web-agent execution even when one external provider is unstable locally.

**Why this priority**: The sandbox provider choice influences the executor roadmap. If the product contract depends on one provider, local provider failure can stall file output, skill execution, and later runtime modules.

**Independent Test**: A contributor can read this specification and clearly identify the short-term provider direction, the experimental OpenSandbox status, and the stable Xelora-owned runtime contract without consulting older design threads.

**Acceptance Scenarios**:

1. **Given** a contributor preparing the runtime plan, **When** they open this specification, **Then** they can see that independent sandbox execution remains the target architecture.
2. **Given** the current OpenSandbox local integration failure, **When** a contributor reads this specification, **Then** they understand that OpenSandbox is retained as an experimental provider rather than the only active path.
3. **Given** a future planning discussion, **When** reviewers use this specification, **Then** they can evaluate Docker executor, E2B, Daytona, OpenSandbox, CubeSandbox, or stronger isolation providers without changing Xelora-owned contracts.

---

### User Story 2 - Preserve Xelora-Owned Product Semantics (Priority: P2)

As the system architect, I want the specification to keep Xelora in control of workspace identity, artifact records, execution policy, and user-visible history even when OpenSandbox is adopted, so that we can reuse a mature sandbox layer without giving up core product ownership.

**Why this priority**: The user explicitly wants to reuse mature open-source modules with minimal invasive modification, but does not want the product to become provider-defined.

**Independent Test**: A reviewer can use this specification alone to distinguish Xelora-owned contracts from OpenSandbox-owned execution mechanics.

**Acceptance Scenarios**:

1. **Given** the OpenSandbox baseline is selected, **When** the reviewer checks this specification, **Then** they can see which responsibilities remain owned by Xelora-facing modules.
2. **Given** a proposed provider integration detail, **When** the reviewer compares it with the specification, **Then** they can reject any design that lets the sandbox provider take ownership of session identity, artifact identity, or user-visible job history.

---

### User Story 3 - Keep The Runtime Extensible Beyond One Sandbox (Priority: P3)

As a maintainer, I want the specification to keep the sandbox provider replaceable as Docker executor, OpenSandbox, E2B, Daytona, CubeSandbox, or stronger isolation backends are evaluated, so that future modules for browser automation, file creation, file editing, or other execution services can evolve without forcing a redesign of the product contract.

**Why this priority**: The user cares more about long-term extensibility than a one-off sandbox integration, and wants later module choices to build on stable boundaries instead of provider lock-in.

**Independent Test**: A contributor can read this specification and derive a roadmap where Controlled Docker Executor is the first usable local provider, OpenSandbox remains experimental, and other reference modules remain independently selectable and replaceable.

**Acceptance Scenarios**:

1. **Given** a future sandbox or execution provider proposal, **When** a maintainer checks this specification, **Then** they can identify the Xelora-facing contracts that must remain stable during replacement.
2. **Given** later work on browser, file, or artifact services, **When** contributors use this specification, **Then** they can plan those modules without assuming that OpenSandbox owns non-sandbox product behavior.

---

### Edge Cases

- OpenSandbox remains architecturally interesting, but the local Docker Desktop integration path fails command execution despite successful sandbox creation.
- The short-term Docker executor validates user-visible product flow before stronger sandbox isolation is available.
- A later provider offers stronger isolation or scaling, but only if Xelora gives up control of workspace persistence or artifact metadata.
- Browser automation and file generation need to run alongside sandbox execution, but the system must avoid coupling those capabilities to one sandbox-specific workflow.
- The first baseline supports execution well but leaves some advanced file or office-editing capabilities to later modules.
- Historical planning documents mention CubeSandbox or OpenSandbox as a primary baseline and could be mistaken for the current active direction if the provider status is not explicit.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The specification MUST define independent sandbox execution as the target architecture for Xelora's web agent runtime.
- **FR-002**: The specification MUST state that OpenSandbox is retained as an experimental provider after local command-proxy failures, not as the only active implementation path.
- **FR-003**: The specification MUST preserve the existing runtime module scope around sandbox execution, execution gateway, session workspace management, artifact management, browser automation, file capability services, and observability or audit support.
- **FR-004**: The specification MUST define which responsibilities remain owned by Xelora-facing modules, including session workspace identity, job identity, artifact records, policy decisions, and user-visible execution history.
- **FR-005**: The specification MUST define which responsibilities may be delegated to Docker executor, OpenSandbox, E2B, Daytona, CubeSandbox, or later replaceable providers, including low-level execution, sandbox lifecycle, runtime resource controls, and other provider-specific mechanics.
- **FR-006**: The specification MUST preserve a provider-agnostic contract boundary so that OpenSandbox can be replaced later without redesigning Xelora-owned product semantics.
- **FR-007**: The specification MUST define the short-term controlled Docker execution service as the preferred local validation provider while stronger isolation providers remain replaceable options.
- **FR-008**: The specification MUST keep real file artifacts, not text-only responses, as a first-class runtime outcome for later file capability modules.
- **FR-009**: The specification MUST support future browser automation, spreadsheet handling, report generation, presentation generation, and other file-oriented capability modules without requiring them to be owned by the sandbox provider.
- **FR-010**: The specification MUST define an adoption sequence that distinguishes the first usable runtime baseline from later supporting modules.
- **FR-011**: The specification MUST make clear that mature external modules should be adapted with minimal invasive modification whenever that does not compromise Xelora-owned contracts.
- **FR-012**: The specification MUST identify mature reference projects by module family, including OpenHands Software Agent SDK for workspace semantics, E2B and Daytona for sandbox API models, gVisor/Kata/Firecracker for isolation, Gotenberg for PDF conversion, SheetJS and Univer for spreadsheets, PptxGenJS for presentation generation, and ONLYOFFICE for advanced office editing.

### Key Entities

- **Sandbox Provider**: A replaceable execution backend used by Xelora through the executor gateway.
- **Local Docker Executor**: A controlled local provider that creates an isolated working directory, runs commands in a managed container, captures logs, and returns artifacts through Xelora-owned contracts.
- **Experimental Provider**: A provider retained for continued evaluation but not allowed to block the first usable runtime path.
- **Runtime Module Family**: A major architectural capability area such as sandbox execution, browser automation, file capability, artifact management, or observability.
- **Xelora-owned Contract**: A product-facing responsibility that must remain stable regardless of which provider is active behind the gateway.
- **Provider-owned Capability**: A replaceable implementation concern such as low-level execution, sandbox lifecycle, browser control, or format-specific conversion mechanics.
- **Adoption Stage**: A planned phase that introduces one or more runtime module families while preserving the stable Xelora-facing contract.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A contributor can identify the replaceable-provider strategy and the short-term Docker executor direction within 5 minutes of reading the specification.
- **SC-002**: A reviewer can distinguish Xelora-owned responsibilities from provider-owned responsibilities for every module family in initial scope using the specification alone.
- **SC-003**: The next planning pass can derive a staged implementation roadmap without reopening the question of whether OpenSandbox must be the only active baseline.
- **SC-004**: The specification leaves no ambiguity about whether artifact creation, file output, and execution history remain first-class product outcomes owned by Xelora-facing modules.
- **SC-005**: A future provider replacement discussion can use this specification to evaluate compatibility without redefining workspace identity, artifact identity, or policy ownership.

## Assumptions

- The primary audience is Xelora maintainers and contributors planning runtime architecture and staged implementation work.
- OpenSandbox is useful as a mature reference and experimental provider, but the current local integration result does not justify blocking the product roadmap on it.
- A controlled Docker executor can validate the Xelora-owned contract before stronger provider isolation is selected.
- Xelora should continue to own user-facing runtime semantics even when mature external execution or file-capability modules are reused.
- The sandbox service may run independently from the main Xelora web deployment if that leads to simpler operations, better resource control, or easier future replacement.
- Later design artifacts will update existing CubeSandbox-oriented plans, task lists, and execution notes to align with this new baseline.
