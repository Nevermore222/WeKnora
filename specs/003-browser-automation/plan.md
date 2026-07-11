# Implementation Plan: Browser Automation Provider Path

**Branch**: `003-browser-automation` | **Date**: 2026-07-10 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/003-browser-automation/spec.md`

## Summary

Add a browser automation provider path that lets web agents navigate to URLs and capture screenshots or page content as real artifacts. The implementation reuses the existing executor gateway, workspace binding, and artifact model. A new `BrowserProvider` interface parallels the sandbox `Provider` interface. A new `browser_navigate` agent tool dispatches browser tasks through the gateway. The first browser provider runs a headless browser snapshot script inside the controlled Docker sandbox as a preloaded skill.

## Technical Context

**Language/Version**: Go 1.26 backend, Vue 3 + TypeScript frontend, Python for the browser snapshot skill script

**Primary Dependencies**: Existing `internal/executor` gateway and provider registry, `internal/sandbox` Docker execution, `internal/agent/tools` tool registry, Playwright or Selenium for headless browser automation inside the sandbox

**Storage**: No new database tables; browser job and artifact records are transient per-execution structures returned to the agent tool. Persistent job history is a later observability task (T-013).

**Testing**: Go unit tests for the gateway `RunBrowserTaskJob` method and browser provider seam; manual smoke validation through the web UI and a `browser-smoke.ps1` script

**Target Platform**: Dockerized web application running on Linux/Docker Desktop with the existing controlled Docker sandbox

**Project Type**: Web application with backend + frontend

**Performance Goals**: Browser navigation tasks for a typical public page complete within 30 seconds; screenshot or page content artifact is registered and downloadable immediately after job completion

**Constraints**: Must reuse the existing workspace binding and boundary enforcement; must not leak browser provider specifics into session or artifact contracts; must preserve compatibility for conversations without browser tasks

**Scale/Scope**: Single-page navigation, screenshot capture, and page content capture in v1; multi-step scripted browser workflows and form interaction are later phases

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

No project constitution has been initialized at `.specify/memory/constitution.md`, so there are no constitution gates to enforce yet.

Working rules applied for this plan:

- Reuse the existing gateway, workspace, and artifact model instead of introducing a parallel browser system.
- Keep the browser provider behind a replaceable interface seam.
- Preserve compatibility for conversations that never use browser tasks.

## Project Structure

### Documentation (this feature)

```text
specs/003-browser-automation/
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/
|   `-- browser-automation.openapi.yaml
`-- checklists/
    `-- requirements.md
```

### Source Code (repository root)

```text
internal/
|-- executor/
|   |-- gateway.go          # Add RunBrowserTaskJob + browser provider registry
|   |-- browser_types.go    # NEW: BrowserJobRequest, BrowserJob, BrowserTaskResult, BrowserProvider interface
|   |-- browser_provider.go # NEW: ControlledDockerBrowserProvider implementation
|   |-- gateway_test.go     # Add browser task tests
|-- agent/
|   |-- tools/
|   |   |-- browser_navigate.go    # NEW: browser_navigate agent tool
|   |   |-- definitions.go         # Register tool name
|   |   |-- registry.go            # Register tool
skills/preloaded/
|-- browser-snapshot/
|   |-- SKILL.md            # Skill definition
|   `-- scripts/
|       `-- browser_snapshot.py  # Headless browser navigation + capture script
scripts/
|-- browser-smoke.ps1      # NEW: smoke test for browser task flow
```

**Structure Decision**: Keep this feature inside existing executor and agent-tool modules. The gateway gains a `RunBrowserTaskJob` method and a browser provider registry. The browser snapshot script is a preloaded skill, consistent with how `officecli-document-editing` and `workspace-file-writer` work.

## Phase 0: Research

See [research.md](./research.md).

Primary decisions:

1. Browser tasks dispatch through the existing `Gateway` via `RunBrowserTaskJob`.
2. A `BrowserProvider` interface parallels the sandbox `Provider` interface.
3. Browser artifacts are detected by the existing file-snapshot diffing.
4. The first browser provider runs inside the controlled Docker sandbox as a preloaded skill.
5. A new `browser_navigate` agent tool invokes browser tasks through the gateway.
6. Browser tasks reuse the configurable timeout system with a browser-specific override.

## Phase 1: Design

See:

- [data-model.md](./data-model.md)
- [contracts/browser-automation.openapi.yaml](./contracts/browser-automation.openapi.yaml)
- [quickstart.md](./quickstart.md)

Design outcomes:

1. `BrowserJobRequest` and `BrowserJob` types parallel the skill job types.
2. `BrowserProvider` interface is structurally parallel to the sandbox provider interface.
3. The gateway resolves the conversation output context and routes browser artifacts to the bound workspace.
4. Browser provider errors are surfaced as structured job failures.

## Implementation Strategy

### Backend

1. Define `BrowserJobRequest`, `BrowserJob`, `BrowserTaskResult`, and `BrowserProvider` interface in `internal/executor/browser_types.go`.
2. Add `RunBrowserTaskJob` to the gateway with workspace resolution, boundary checks, provider selection, execution, and artifact detection.
3. Implement `ControlledDockerBrowserProvider` in `internal/executor/browser_provider.go` that runs the browser snapshot script via the existing sandbox Docker execution.
4. Add the `browser_navigate` agent tool in `internal/agent/tools/browser_navigate.go`.
5. Register the tool in the agent tool registry.
6. Add gateway tests for browser task routing, boundary enforcement, and provider error handling.

### Skill

1. Create the `skills/preloaded/browser-snapshot/` skill with a `SKILL.md`.
2. Write `scripts/browser_snapshot.py` that takes a URL and capture mode, launches a headless browser, navigates, and writes screenshot/content files to the working directory.
3. Ensure the sandbox Docker image has the browser binary installed (or add a browser-capable image reference).

### Verification

1. A browser navigation task from an agent-enabled conversation produces a registered screenshot artifact.
2. The artifact is stored inside the bound workspace when a binding exists.
3. Browser provider errors surface as structured failures.
4. Unbound conversations fall back to the skill-private path, consistent with skill jobs.
5. Legacy conversations without browser tasks remain unaffected.

## Complexity Tracking

No constitution violations or exceptional complexity exemptions are required for this feature.

## Risks And Mitigations

| Risk | Mitigation |
|------|------------|
| The sandbox Docker image lacks a browser binary | Add Playwright/Selenium to the sandbox image build or use a browser-capable image reference |
| Headless browser launch is slow or flaky in Docker | Start with single-page navigation; keep timeout configurable; add retry in later phase |
| Large screenshots exceed storage or preview limits | Add size-based filtering in artifact detection in a later phase |
| The browser provider interface diverges from the sandbox provider | Keep both interfaces structurally parallel and select from a shared registry |
| Agent invokes browser tasks on URLs that require authentication | Document the limitation; return partial results with clear messaging |
