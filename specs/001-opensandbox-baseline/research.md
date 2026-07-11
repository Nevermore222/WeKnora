# Research: Xelora Replaceable Sandbox Runtime Baseline

## Decision: Keep independent sandbox execution but avoid single-provider lock-in

**Rationale**: Xelora still needs an independently managed execution layer that the web product can call and orchestrate. Recent local validation showed that OpenSandbox can start and create sandboxes under Docker Desktop, but its command proxy path returns `502` for simple command execution. That makes it too risky as the only active implementation path. The product should keep the executor gateway and provider abstraction, use a controlled Docker executor for the first usable local path, and retain OpenSandbox as an experimental provider.

**Alternatives considered**:

- OpenSandbox as the only first baseline: attractive API surface, but current local command execution is not stable enough to block the roadmap.
- CubeSandbox as the first baseline: strong security direction, but pushes local setup toward KVM and host readiness concerns.
- Hosted E2B-first path: mature API model, but less aligned with local self-managed deployment goals.
- Local provider stub only: valuable for gateway validation, but insufficient as the long-term isolation story.

## Decision: Use a controlled Docker executor as the first usable local provider

**Rationale**: The short-term product risk is not perfect isolation; it is that web agents cannot reliably create or modify real files. A controlled Docker executor can validate Xelora-owned session workspaces, job lifecycle, logs, artifacts, and file outputs while stronger sandbox providers remain replaceable. It should be treated as the local validation provider, not the final multi-tenant security boundary.

**Alternatives considered**:

- Wait for OpenSandbox to be fixed before building product flow: preserves one external-provider path, but blocks file output and skill execution progress.
- Run scripts in the app container: simpler, but weakens isolation boundaries and mixes product runtime with execution runtime.
- Build a full microVM stack immediately: stronger isolation, but too much operational complexity for the first local validation stage.

## Decision: Keep Xelora as the owner of product contracts

**Rationale**: The user wants mature external modules reused with minimal invasive modification, but does not want core product semantics outsourced. Therefore Xelora must remain the authority for session workspace identity, job identity, artifact identity, policy enforcement, provider routing, and user-visible execution history. OpenSandbox is the first execution substrate, not the owner of product behavior.

**Alternatives considered**:

- Provider-owned workspaces or artifacts: simpler short term, but creates lock-in and makes browser, file, and audit modules harder to evolve independently.
- Fully custom runtime stack: preserves ownership, but ignores the user's explicit preference to stand on mature open-source components.

## Decision: Treat OpenSandbox as an experimental independently operated provider

**Rationale**: OpenSandbox still has useful reference value: it exposes sandbox lifecycle, command, file, SDK, CLI, and MCP concepts that map well to Xelora's desired runtime. However, current Docker Desktop validation shows that it should not be the only active path. Keeping it behind the provider interface preserves future optionality without letting provider-specific behavior leak into product contracts.

**Alternatives considered**:

- Remove OpenSandbox entirely: loses a useful mature reference and future provider candidate.
- Keep OpenSandbox as primary despite failing smoke tests: risks stalling the project on third-party integration details.
- Make the execution provider purely in-process: easier to prototype, but not aligned with broader runtime isolation.

## Decision: Preserve OpenHands Software Agent SDK as the semantic reference for agent workspaces

**Rationale**: OpenHands Software Agent SDK remains a strong semantic reference for workspace-oriented agent behavior and code-task execution concepts. It is still more appropriate as a guidance layer for workspace semantics than as the first sandbox substrate itself.

**Alternatives considered**:

- Treat OpenSandbox as both execution substrate and agent-semantic source: this would blur low-level runtime responsibilities with higher-level product workflow decisions.
- Design workspace semantics from scratch: unnecessary given the reference material already available.

## Decision: Keep browser automation and file capability modules independent from the sandbox baseline

**Rationale**: Xelora needs to support real file outputs and later file modifications such as Markdown, spreadsheets, reports, presentations, and PDF flows. Those modules should be orchestrated through shared artifact and job contracts, not hidden inside one sandbox-specific integration. The existing reference split remains valid: agent-browser for browser automation, Gotenberg for service-style PDF conversion, SheetJS for spreadsheet mechanics, Univer for embedded office-style surfaces, and PptxGenJS for generated presentation output.

**Alternatives considered**:

- Let the sandbox provider own file workflows end to end: this would reduce flexibility and make later module replacement harder.
- Choose a single universal office stack for every output and editing need: too heavy and likely to distort module boundaries too early.

## Decision: Track mature providers by module family

**Rationale**: Xelora's runtime is not one module. The mature-project references should be split by responsibility so each can be wrapped with minimal invasive modification. OpenHands guides workspace semantics; E2B and Daytona guide sandbox API and developer-environment product shape; gVisor, Kata Containers, and Firecracker guide isolation hardening; Gotenberg, SheetJS, Univer, PptxGenJS, and ONLYOFFICE guide file output and editing.

**Alternatives considered**:

- Search for a single universal platform: likely to overfit Xelora to another product's semantics.
- Build all file and sandbox mechanics from scratch: maximizes control but ignores mature, battle-tested modules.
- Put office, browser, and execution tasks inside one sandbox provider: simpler wiring early, but harder replacement and poorer product ownership later.

## Decision: Preserve a staged adoption roadmap

**Rationale**: The runtime scope is still broad. The first useful baseline should focus on the execution gateway, session workspace ownership, artifact-first outcomes, and the controlled Docker executor. OpenSandbox remains an experimental provider. File services, browser automation, and richer observability should follow in later stages.

**Alternatives considered**:

- Implement all runtime modules at once: too broad and likely to stall.
- Re-scope everything around only the sandbox provider swap: too narrow and would fail to prepare the later browser and file work the user already cares about.
