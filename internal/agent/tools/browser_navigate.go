package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tencent/Xelora/internal/agent/skills"
	"github.com/Tencent/Xelora/internal/executor"
	"github.com/Tencent/Xelora/internal/logger"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/utils"
)

var browserNavigateTool = BaseTool{
	name: ToolBrowserNavigate,
	description: `Open a browser and navigate to a URL, capturing a screenshot and/or page content.

## Usage
- Use this tool when the agent needs to inspect a web page visually or capture its content
- The screenshot and page content are saved as artifacts in the conversation workspace
- The browser runs inside the controlled Docker sandbox for isolation

## When to Use
- When the user asks to visit, open, or check a website
- When the agent needs to see what a page looks like
- When page content or screenshots are needed as conversation artifacts

## Returns
- Screenshot and/or page content artifact paths
- Job status and execution summary
- Artifact list with download references`,
	schema: utils.GenerateSchema[BrowserNavigateInput](),
}

// BrowserNavigateInput defines the input for the browser_navigate tool.
type BrowserNavigateInput struct {
	URL         string `json:"url" jsonschema:"Target page URL to navigate to"`
	CaptureMode string `json:"capture_mode,omitempty" jsonschema:"What to capture: screenshot, content, or both. Defaults to screenshot."`
	Provider    string `json:"provider,omitempty" jsonschema:"Optional browser provider key. Defaults to controlled-docker-browser."`
}

// BrowserNavigateTool allows the agent to dispatch browser navigation tasks
// through the executor gateway. It mirrors ExecuteSkillScriptTool in structure
// but uses the gateway's RunBrowserTaskJob method.
type BrowserNavigateTool struct {
	BaseTool
	gateway                  *executor.Gateway
	skillExecutor            executor.SkillExecutor
	sessionWorkspaceResolver func(sessionID string) *types.SessionWorkspaceBinding
}

func NewBrowserNavigateTool() *BrowserNavigateTool {
	return &BrowserNavigateTool{
		BaseTool: browserNavigateTool,
		gateway:  executor.NewGateway(),
	}
}

// WithSkillManager attaches the skill manager so the browser-snapshot skill
// script can be prepared and executed through the same skill infrastructure
// as file-producing skills.
func (t *BrowserNavigateTool) WithSkillManager(mgr *skills.Manager) *BrowserNavigateTool {
	t.skillExecutor = mgr
	return t
}

// WithSessionWorkspaceResolver attaches a resolver that lets the tool look
// up the session's bound workspace so browser artifacts are routed correctly.
func (t *BrowserNavigateTool) WithSessionWorkspaceResolver(resolver func(sessionID string) *types.SessionWorkspaceBinding) *BrowserNavigateTool {
	t.sessionWorkspaceResolver = resolver
	return t
}

// WithGateway allows injecting a pre-configured gateway, useful for testing.
func (t *BrowserNavigateTool) WithGateway(gateway *executor.Gateway) *BrowserNavigateTool {
	t.gateway = gateway
	return t
}

func (t *BrowserNavigateTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	logger.Infof(ctx, "[Tool][BrowserNavigate] Execute started")

	var input BrowserNavigateInput
	if err := json.Unmarshal(args, &input); err != nil {
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse args: %v", err),
		}, nil
	}

	if input.URL == "" {
		return &types.ToolResult{
			Success: false,
			Error:   "url is required",
		}, nil
	}

	captureMode := executor.BrowserCaptureScreenshot
	switch strings.ToLower(strings.TrimSpace(input.CaptureMode)) {
	case "content":
		captureMode = executor.BrowserCaptureContent
	case "both":
		captureMode = executor.BrowserCaptureBoth
	}

	var jobRequest executor.BrowserJobRequest
	if meta, ok := ToolExecFromContext(ctx); ok {
		jobRequest.SessionID = meta.SessionID
		jobRequest.AssistantMessageID = meta.AssistantMessageID
		jobRequest.RequestID = meta.RequestID
		jobRequest.ToolCallID = meta.ToolCallID
	}
	jobRequest.Provider = strings.TrimSpace(input.Provider)
	jobRequest.URL = input.URL
	jobRequest.CaptureMode = captureMode

	if t.sessionWorkspaceResolver != nil && jobRequest.SessionID != "" {
		jobRequest.WorkspaceBinding = t.sessionWorkspaceResolver(jobRequest.SessionID)
	}

	logger.Infof(ctx, "[Tool][BrowserNavigate] Navigating to: %s, capture: %s", input.URL, captureMode)

	skillExecutor := t.getSkillExecutor()
	if skillExecutor == nil {
		return &types.ToolResult{
			Success: false,
			Error:   "Skills are not enabled; browser navigation requires the skill manager",
		}, nil
	}

	execution, err := t.gateway.RunBrowserTaskJob(ctx, jobRequest, skillExecutor)
	if err != nil {
		logger.Errorf(ctx, "[Tool][BrowserNavigate] Browser task failed: %v", err)
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Browser task failed: %v", err),
		}, nil
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("=== Browser Navigation: %s ===\n\n", input.URL))
	builder.WriteString(fmt.Sprintf("**Capture Mode**: %s\n", captureMode))
	builder.WriteString(fmt.Sprintf("**Exit Code**: %d\n", execution.Job.ExitCode))
	builder.WriteString(fmt.Sprintf("**Duration**: %dms\n\n", execution.Job.DurationMs))

	if execution.Job.Error != "" {
		builder.WriteString("## Error\n\n")
		builder.WriteString(execution.Job.Error)
		builder.WriteString("\n")
	}

	if len(execution.Artifacts) > 0 {
		builder.WriteString("\n## Artifacts\n\n")
		for _, artifact := range execution.Artifacts {
			builder.WriteString(fmt.Sprintf("- `%s` (%s, %s, %d bytes)\n", artifact.RelativePath, artifact.Kind, artifact.ChangeType, artifact.Size))
		}
	} else {
		builder.WriteString("\n## Artifacts\n\n")
		builder.WriteString("No output artifact was detected.\n")
	}

	success := execution.Result.IsSuccess()

	resultData := map[string]interface{}{
		"url":               input.URL,
		"capture_mode":      string(captureMode),
		"workspace":         execution.Workspace,
		"provider":          execution.Provider,
		"job":               execution.Job,
		"artifacts":         execution.Artifacts,
		"artifact_detected": execution.ArtifactDetected,
		"exit_code":         execution.Job.ExitCode,
	}

	if execution.Result != nil {
		resultData["stdout"] = execution.Result.Stdout
		resultData["stderr"] = execution.Result.Stderr
	}

	return &types.ToolResult{
		Success: success,
		Output:  builder.String(),
		Data:    resultData,
		Error: func() string {
			if !success {
				if execution.Job.Error != "" {
					return execution.Job.Error
				}
				return fmt.Sprintf("Browser task exited with code %d", execution.Job.ExitCode)
			}
			return ""
		}(),
	}, nil
}

// getSkillExecutor returns the skill executor interface for browser task
// dispatch. The browser-snapshot skill script runs through the same skill
// manager as file-producing skills.
func (t *BrowserNavigateTool) getSkillExecutor() executor.SkillExecutor {
	return t.skillExecutor
}

func (t *BrowserNavigateTool) Cleanup(ctx context.Context) error {
	return nil
}
