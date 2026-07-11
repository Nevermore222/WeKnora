# Phase 0 Research: Browser Automation Provider Path

**Branch**: `003-browser-automation` | **Date**: 2026-07-10

## Research Questions

### RQ1: How should browser tasks integrate with the existing executor gateway?

**Decision**: Browser tasks dispatch through the same `Gateway` as skill script jobs, using a new `BrowserJobRequest` type. The gateway already owns workspace selection, artifact detection, and workspace binding resolution. Browser tasks reuse this flow; only the provider execution path differs.

**Rationale**: The gateway's `RunSkillScriptJob` already resolves the `ConversationOutputContext` from the session workspace binding, enforces boundary checks, and detects artifacts by file-snapshot diffing. A browser task follows the same lifecycle: prepare workspace, dispatch to provider, snapshot for new artifacts, register them. Adding a parallel gateway would duplicate workspace binding and artifact ownership logic.

**Alternative rejected**: A separate `BrowserGateway` type. This would fragment artifact registration and workspace routing, violating the "same artifact model" requirement (FR-003, SC-003).

### RQ2: Should the browser provider implement the existing `Provider` interface or a new one?

**Decision**: Add a `BrowserProvider` interface that is structurally parallel to the existing `Provider` interface but carries browser-specific request and result types. The gateway selects providers by name from a shared registry, and browser tasks use the same `selectProvider` helper.

**Rationale**: The existing `Provider.ExecuteSkillScript` takes `SkillJobRequest` and `*skills.PreparedScriptExecution`, which are script-oriented. Browser tasks have different inputs (URL, action, capture mode) and outputs (screenshot bytes, page content). Forcing them through the skill-script interface would require awkward encoding. A parallel `BrowserProvider` interface keeps each provider type clean while sharing the same gateway-level workspace, artifact, and job-state machinery.

**Alternative rejected**: Extending `Provider` with a `ExecuteBrowserTask` method. This would bloat every sandbox provider with an unused browser method, violating the single-responsibility principle.

### RQ3: How should browser artifacts (screenshots, page content) be stored and registered?

**Decision**: The browser provider writes screenshot files (PNG) and/or page content files (HTML/Markdown) to the workspace root. The gateway's existing `snapshotFiles` / `detectArtifacts` logic then registers them as `ArtifactRecord` entries with the correct workspace, job, and session IDs. The artifact kind detection in `detectArtifactKind` already handles `.png`, `.jpg`, `.html`, and `.md` extensions.

**Rationale**: Reusing the file-snapshot diffing approach means browser artifacts are detected identically to skill-produced artifacts. No parallel artifact registration path is needed.

**Alternative rejected**: Having the browser provider register artifacts directly. This would bypass the gateway's boundary checks and workspace ownership, violating FR-007.

### RQ4: What is the first browser provider reference?

**Decision**: Use a headless Chrome/Chromium approach via a Python script that runs inside the existing controlled Docker sandbox, consistent with the current sandbox execution model. The script uses a CDP-compatible library (e.g., Playwright or Selenium) to launch a headless browser, navigate to the URL, capture a screenshot, and optionally capture page content. The script is preloaded as a skill under `skills/preloaded/browser-snapshot/`.

**Rationale**: The controlled Docker executor already runs Python scripts with the workspace mounted. A browser snapshot skill fits the same pattern as the existing `officecli-document-editing` and `workspace-file-writer` skills. The browser binary runs inside the sandbox image, not on the Xelora host.

**Alternative rejected**: Running the browser on the host or in a separate browser service. This would add operational complexity and bypass the existing sandbox isolation. A later phase can evaluate a dedicated browser service if the sandbox-based approach proves limiting.

### RQ5: How does the agent invoke browser tasks?

**Decision**: Add a new agent tool `browser_navigate` that takes a URL and optional capture mode (screenshot, content, or both). The tool constructs a `BrowserJobRequest`, calls the gateway's new `RunBrowserTaskJob` method, and returns the result with artifact references, mirroring how `ExecuteSkillScriptTool` works today.

**Rationale**: The agent tool registry (`internal/agent/tools/registry.go`) already discovers and registers tools. A dedicated browser tool is cleaner than overloading `execute_skill_script` with browser semantics.

**Alternative rejected**: Routing browser tasks through `execute_skill_script`. This would require the agent to know about a specific skill script path, which is a leaky abstraction.

### RQ6: What timeout and error handling should browser tasks use?

**Decision**: Browser tasks use the same configurable timeout as sandbox execution (`XELORA_SANDBOX_TIMEOUT`), defaulting to 60 seconds. A configurable env var `XELORA_BROWSER_TIMEOUT` overrides this for browser tasks when set. Timeouts produce a structured job failure with `Error` set to a timeout message, consistent with `sandbox.ExecuteResult` behavior.

**Rationale**: Reusing the existing timeout configuration avoids a new settings surface while allowing browser-specific tuning.

**Alternative rejected**: A fixed browser timeout. Different pages and network conditions need different timeout tolerances.

## Summary of Decisions

1. Browser tasks dispatch through the existing `Gateway` via a new `RunBrowserTaskJob` method.
2. A `BrowserProvider` interface parallels the sandbox `Provider` interface.
3. Browser artifacts are detected by the existing file-snapshot diffing and registered with the same `ArtifactRecord` model.
4. The first browser provider runs inside the controlled Docker sandbox as a preloaded skill script using a CDP-compatible library.
5. A new `browser_navigate` agent tool invokes browser tasks through the gateway.
6. Browser tasks reuse the configurable timeout system with an optional browser-specific override.
