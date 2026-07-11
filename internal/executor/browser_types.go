package executor

import (
	"context"
	"time"

	"github.com/Tencent/Xelora/internal/sandbox"
	"github.com/Tencent/Xelora/internal/types"
)

// BrowserCaptureMode selects what a browser task captures from the page.
type BrowserCaptureMode string

const (
	BrowserCaptureScreenshot BrowserCaptureMode = "screenshot"
	BrowserCaptureContent    BrowserCaptureMode = "content"
	BrowserCaptureBoth       BrowserCaptureMode = "both"
)

// BrowserProviderName constants identify registered browser providers.
const (
	ControlledDockerBrowserProviderName = "controlled-docker-browser"
)

// BrowserJobRequest is the request for a browser navigation task dispatched
// through the gateway. It parallels SkillJobRequest but carries
// browser-specific inputs (URL, capture mode) instead of a skill script path.
type BrowserJobRequest struct {
	SessionID          string
	AssistantMessageID string
	RequestID          string
	ToolCallID         string
	Provider           string
	URL                string
	CaptureMode        BrowserCaptureMode
	Input              string
	WorkspaceBinding   *types.SessionWorkspaceBinding
	Timeout            time.Duration
}

// BrowserJob is the durable job record for a browser task, stored in the same
// job model as skill jobs. Xelora owns the job identity, status, and artifact
// linkage; the browser provider owns only the browser mechanics.
type BrowserJob struct {
	ID                 string          `json:"id"`
	WorkspaceID        string          `json:"workspace_id"`
	SessionID          string          `json:"session_id,omitempty"`
	AssistantMessageID string          `json:"assistant_message_id,omitempty"`
	RequestID          string          `json:"request_id,omitempty"`
	ToolCallID         string          `json:"tool_call_id,omitempty"`
	URL                string          `json:"url"`
	CaptureMode        BrowserCaptureMode `json:"capture_mode"`
	Provider           string          `json:"provider"`
	WorkingDir         string          `json:"working_dir"`
	Status             JobStatus       `json:"status"`
	ExitCode           int             `json:"exit_code"`
	StartedAt          time.Time       `json:"started_at"`
	FinishedAt         time.Time       `json:"finished_at,omitempty"`
	DurationMs         int64           `json:"duration_ms"`
	StdoutSummary      string          `json:"stdout_summary,omitempty"`
	StderrSummary      string          `json:"stderr_summary,omitempty"`
	ArtifactCount      int             `json:"artifact_count"`
	Error              string          `json:"error,omitempty"`
}

// BrowserTaskResult is the outcome returned by the browser provider after
// execution. The provider writes screenshot and/or content files to the
// output directory; the gateway detects them via file-snapshot diffing.
type BrowserTaskResult struct {
	Result        *sandbox.ExecuteResult
	ScreenshotPath string
	ContentPath   string
	PageTitle     string
	FinalURL      string
}

// BrowserProvider is the replaceable interface seam for browser automation
// backends. It parallels the sandbox Provider interface but carries
// browser-specific request and result types. The provider must not become the
// source of truth for job identity, artifact metadata, or workspace ownership.
type BrowserProvider interface {
	Name() string
	ExecuteBrowserTask(ctx context.Context, req BrowserJobRequest, outputDir string, executor SkillExecutor) (*BrowserTaskResult, error)
	Capability(ctx context.Context) ProviderCapability
}

// BrowserTaskExecution is the composite result returned by the gateway,
// containing the job, workspace, provider, and artifacts. It mirrors
// SkillJobExecution so browser artifacts and skill artifacts are consumed
// identically by the agent tool layer.
type BrowserTaskExecution struct {
	Job              *BrowserJob
	Workspace        *WorkspaceRecord
	Provider         ProviderCapability
	Artifacts        []*ArtifactRecord
	ArtifactDetected bool
	Result           *sandbox.ExecuteResult
}
