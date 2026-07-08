# Research: Xelora Agent Runtime Reference Architecture

## Decision: Use CubeSandbox as the first sandbox execution baseline

**Rationale**: CubeSandbox describes itself as a high-performance secure sandbox service for AI agents, built on RustVMM and KVM, with single-node and multi-node deployment support plus E2B SDK compatibility. That matches Xelora's need for a real execution substrate rather than an ad hoc command runner. It also aligns with the user's preference to run the sandbox independently from the main Xelora Docker deployment and, for local work, to use WSL/Linux where virtualization prerequisites are available. [Source](https://github.com/tencentcloud/CubeSandbox) [Source](https://github.com/TencentCloud/CubeSandbox/blob/master/docs/guide/quickstart.md)

**Alternatives considered**:

- E2B: Strong API model, but not the preferred first baseline for this repo.
- Local provider stub: still needed for development flow validation, but not sufficient as the long-term runtime substrate.
- Daytona: strong concepts, but weaker fit as the chosen first baseline for this planning pass.

## Decision: Use OpenHands Software Agent SDK as the semantic reference for agent workspaces

**Rationale**: OpenHands Software Agent SDK is explicitly positioned for building agents that work with code and supports either local workspaces or ephemeral workspaces via Agent Server. That makes it a strong reference for workspace-oriented agent semantics, even though it is not the first sandbox base selected for Xelora. [Source](https://github.com/OpenHands/software-agent-sdk)

**Alternatives considered**:

- Treat CubeSandbox itself as both sandbox and agent semantic reference: this would blur execution substrate concerns with higher-level workflow concerns.
- Design workspace semantics from scratch: unnecessary given the available reference material.

## Decision: Use agent-browser as the primary browser automation reference

**Rationale**: agent-browser is a browser automation CLI for AI agents with a compact agent-oriented interface and CDP/Chrome automation focus. It is a better fit as a replaceable browser provider layer than building browser control from scratch inside Xelora. [Source](https://github.com/vercel-labs/agent-browser)

**Alternatives considered**:

- Generic Playwright-only integration: workable later, but less aligned with the agent-runtime orientation of this architecture spec.
- Embedding browser automation directly into the main app: this would tighten coupling too early.

## Decision: Split file capabilities into service-style and embedded-style references

**Rationale**:

- Gotenberg is a mature Docker-based API for converting documents to PDF and is well suited for service-style conversion. [Source](https://github.com/gotenberg/gotenberg)
- SheetJS is a strong baseline for spreadsheet read/write mechanics.
- Univer is a full-stack framework for spreadsheet/document/presentation experiences inside a product, which makes it strong for embedded editing or preview surfaces, but it is also in heavy development, so it should be adopted thoughtfully. [Source](https://github.com/dream-num/univer) [Source](https://github.com/dream-num/univer/discussions/4754)
- PptxGenJS remains the practical reference for generated presentation output.

**Alternatives considered**:

- OnlyOffice as the universal office solution: too heavy for the first planning baseline.
- A single file provider for every format: too blunt and likely to force poor module boundaries.

## Decision: Keep Xelora as the owner of product contracts

**Rationale**: The user explicitly wants to reuse mature modules without giving up control of core behavior. Therefore Xelora, and especially the future Executor Gateway, must retain ownership of session workspaces, job identity, artifact identity, policy enforcement, and user-visible history. Providers remain replaceable execution or conversion layers.

**Alternatives considered**:

- Provider-owned workspaces or artifacts: simpler at first integration time, but creates long-term lock-in.
- Fully custom runtime implementation: preserves ownership, but ignores the user's explicit preference to build on mature projects.

## Decision: Preserve a staged adoption model

**Rationale**: The module set is too broad to implement safely in one sweep. The first useful baseline is still: execution gateway, session workspace model, local provider validation, CubeSandbox adapter, and artifact-first outcomes. Browser automation and richer file modules should follow after that.

**Alternatives considered**:

- Plan every module as first-phase work: too broad and likely to stall.
- Focus only on CubeSandbox: too narrow and fails to answer the user's broader architectural question.
