# Data Model: Browser Automation Provider Path

**Branch**: `003-browser-automation` | **Date**: 2026-07-10

## New Types

### BrowserJobRequest

The request for a browser navigation task, dispatched through the gateway.

| Field | Type | Description |
|-------|------|-------------|
| SessionID | string | Conversation session ID |
| AssistantMessageID | string | Agent message that triggered the task |
| RequestID | string | Request trace ID |
| ToolCallID | string | Tool invocation ID |
| Provider | string | Browser provider name (defaults to controlled-docker) |
| URL | string | Target page URL |
| CaptureMode | string | What to capture: "screenshot", "content", or "both" |
| WorkspaceBinding | *types.SessionWorkspaceBinding | Conversation workspace binding for output routing |
| Timeout | time.Duration | Maximum page load wait (0 = use provider default) |

### BrowserTaskResult

The outcome returned by the browser provider after execution.

| Field | Type | Description |
|-------|------|-------------|
| Result | *sandbox.ExecuteResult | Execution status (exit code, stdout, stderr, duration, error) |
| ScreenshotPath | string | Absolute path to screenshot file when captured |
| ContentPath | string | Absolute path to page content file when captured |
| PageTitle | string | Title of the loaded page |
| FinalURL | string | Final URL after any redirects |

### BrowserJob

The durable job record for a browser task, stored in the same job model as skill jobs.

| Field | Type | Description |
|-------|------|-------------|
| ID | string | Unique job ID |
| WorkspaceID | string | Workspace the task ran in |
| SessionID | string | Conversation session ID |
| AssistantMessageID | string | Agent message ID |
| RequestID | string | Request trace ID |
| ToolCallID | string | Tool call ID |
| URL | string | Target URL |
| CaptureMode | string | What was captured |
| Provider | string | Browser provider name |
| WorkingDir | string | Workspace root for artifact output |
| Status | JobStatus | running / succeeded / failed |
| ExitCode | int | Script exit code |
| StartedAt | time.Time | Dispatch time |
| FinishedAt | time.Time | Completion time |
| DurationMs | int64 | Execution duration |
| StdoutSummary | string | Truncated stdout |
| StderrSummary | string | Truncated stderr |
| ArtifactCount | int | Number of artifacts produced |
| Error | string | Error message on failure |

### BrowserTaskExecution

The composite result returned by the gateway, containing the job, workspace, provider, and artifacts.

| Field | Type | Description |
|-------|------|-------------|
| Job | *BrowserJob | The browser job record |
| Workspace | *WorkspaceRecord | The workspace used |
| Provider | ProviderCapability | Provider capability snapshot |
| Artifacts | []*ArtifactRecord | Registered artifacts |
| ArtifactDetected | bool | Whether any artifacts were produced |
| Result | *sandbox.ExecuteResult | Raw execution result |

## Existing Types Reused

- `WorkspaceRecord` — reused as-is for browser task workspaces.
- `ArtifactRecord` — reused as-is; browser screenshots and content files are detected by the existing extension-based kind detection.
- `ArtifactKind` — `.png` and `.jpg` map to `ArtifactKindImage`; `.html` and `.md` map to `ArtifactKindMarkdown`.
- `ConversationOutputContext` — reused to resolve bound workspace routing.
- `ProviderCapability` — reused for browser provider capability reporting.
- `JobStatus` — reused for browser job state transitions.

## No Database Changes

Browser tasks do not introduce new database tables or columns. The job and artifact records are transient per-execution structures returned to the agent tool, consistent with how `SkillJobExecution` works today. Persistent job history tracking is a later observability task (T-013).

## Browser Provider Interface

```go
type BrowserProvider interface {
    Name() string
    ExecuteBrowserTask(ctx context.Context, req BrowserJobRequest, outputDir string) (*BrowserTaskResult, error)
    Capability(ctx context.Context) ProviderCapability
}
```

The `BrowserProvider` interface parallels the existing `Provider` interface but carries browser-specific request and result types. The gateway holds a registry of browser providers and selects by name, mirroring `selectProvider` for sandbox providers.

## Browser Provider Environment Variables

| Env Var | Default | Description |
|---------|---------|-------------|
| XELORA_BROWSER_PROVIDER | controlled-docker | Default browser provider name |
| XELORA_BROWSER_TIMEOUT | 60 | Browser task timeout in seconds |
| XELORA_BROWSER_IMAGE | (sandbox image) | Docker image for browser execution |
| XELORA_BROWSER_HEADLESS | true | Whether to run headless mode |
