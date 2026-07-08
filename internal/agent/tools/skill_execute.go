package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Tencent/Xelora/internal/executor"
	"github.com/Tencent/Xelora/internal/agent/skills"
	"github.com/Tencent/Xelora/internal/logger"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/utils"
)

// Tool name constant for execute_skill_script

var executeSkillScriptTool = BaseTool{
	name: ToolExecuteSkillScript,
	description: `Execute a script from a skill in a sandboxed environment.

## Usage
- Use this tool to run utility scripts bundled with a skill
- Scripts are executed in an isolated sandbox for security
- Only scripts from loaded skills can be executed

## When to Use
- When a skill's instructions reference a utility script (e.g., "Run scripts/analyze_form.py")
- When automation or data processing is needed as part of skill workflow
- For deterministic operations where script execution is more reliable than generating code
- When a skill says it creates or updates files, you must actually run the script before claiming success

## Security
- Scripts run in a sandboxed environment with limited permissions
- Network access is disabled by default
- File access is restricted to the skill directory

## Returns
- Script stdout and stderr output
- Exit code indicating success (0) or failure (non-zero)`,
	schema: utils.GenerateSchema[ExecuteSkillScriptInput](),
}

// ExecuteSkillScriptInput defines the input parameters for the execute_skill_script tool
type ExecuteSkillScriptInput struct {
	SkillName  string   `json:"skill_name" jsonschema:"Name of the skill containing the script"`
	ScriptPath string   `json:"script_path" jsonschema:"Relative path to the script within the skill directory (e.g. scripts/analyze.py)"`
	Args       []string `json:"args,omitempty" jsonschema:"Optional command-line arguments to pass to the script. If the script works on a markdown/document file, include that relative file path as a normal positional argument. Flags like --no-quotes are not file paths."`
	Input      string   `json:"input,omitempty" jsonschema:"Optional document content to materialize into a real file before execution when needed. Do not pass shell commands here. If no file path is provided in args, the backend will create a .md file automatically and pass its path to the script."`
}

// ExecuteSkillScriptTool allows the agent to execute skill scripts in a sandbox
type ExecuteSkillScriptTool struct {
	BaseTool
	skillManager *skills.Manager
	gateway      *executor.Gateway
}

// NewExecuteSkillScriptTool creates a new execute_skill_script tool instance
func NewExecuteSkillScriptTool(skillManager *skills.Manager) *ExecuteSkillScriptTool {
	return &ExecuteSkillScriptTool{
		BaseTool:     executeSkillScriptTool,
		skillManager: skillManager,
		gateway:      executor.NewGateway(),
	}
}

// Execute executes the execute_skill_script tool
func (t *ExecuteSkillScriptTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	logger.Infof(ctx, "[Tool][ExecuteSkillScript] Execute started")

	// Parse input
	var input ExecuteSkillScriptInput
	if err := json.Unmarshal(args, &input); err != nil {
		var parseErr error
		input, parseErr = parseExecuteSkillScriptInput(args)
		if parseErr != nil {
			logger.Errorf(ctx, "[Tool][ExecuteSkillScript] Failed to parse args: %v", parseErr)
			return &types.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("Failed to parse args: %v", parseErr),
			}, nil
		}
	}

	// Validate required fields
	if input.SkillName == "" {
		return &types.ToolResult{
			Success: false,
			Error:   "skill_name is required",
		}, nil
	}

	if input.ScriptPath == "" {
		return &types.ToolResult{
			Success: false,
			Error:   "script_path is required",
		}, nil
	}

	input.Args, input.Input = normalizeExecuteSkillPayload(input.Args, input.Input)

	// Check if skill manager is available
	if t.skillManager == nil || !t.skillManager.IsEnabled() {
		return &types.ToolResult{
			Success: false,
			Error:   "Skills are not enabled",
		}, nil
	}

	var jobRequest executor.SkillJobRequest
	if meta, ok := ToolExecFromContext(ctx); ok {
		jobRequest.SessionID = meta.SessionID
		jobRequest.AssistantMessageID = meta.AssistantMessageID
		jobRequest.RequestID = meta.RequestID
		jobRequest.ToolCallID = meta.ToolCallID
	}
	jobRequest.SkillName = input.SkillName
	jobRequest.ScriptPath = input.ScriptPath
	jobRequest.Args = append([]string(nil), input.Args...)
	jobRequest.Input = input.Input

	// Execute the script through the Xelora-owned gateway layer.
	logger.Infof(ctx, "[Tool][ExecuteSkillScript] Executing script: %s/%s with args: %v, input length: %d",
		input.SkillName, input.ScriptPath, input.Args, len(input.Input))

	jobExecution, err := t.gateway.RunSkillScriptJob(ctx, jobRequest, t.skillManager)
	if err != nil {
		logger.Errorf(ctx, "[Tool][ExecuteSkillScript] Script execution failed: %v", err)
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Script execution failed: %v", err),
		}, nil
	}
	result := jobExecution.Result

	// Build output
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("=== Script Execution: %s/%s ===\n\n", input.SkillName, input.ScriptPath))

	if len(input.Args) > 0 {
		builder.WriteString(fmt.Sprintf("**Arguments**: %v\n", input.Args))
	}

	builder.WriteString(fmt.Sprintf("**Exit Code**: %d\n", result.ExitCode))
	builder.WriteString(fmt.Sprintf("**Duration**: %v\n\n", result.Duration))

	if result.Killed {
		builder.WriteString("**Warning**: Script was terminated (timeout or killed)\n\n")
	}

	if result.Stdout != "" {
		builder.WriteString("## Standard Output\n\n")
		builder.WriteString("```\n")
		builder.WriteString(result.Stdout)
		if !strings.HasSuffix(result.Stdout, "\n") {
			builder.WriteString("\n")
		}
		builder.WriteString("```\n\n")
	}

	if result.Stderr != "" {
		builder.WriteString("## Standard Error\n\n")
		builder.WriteString("```\n")
		builder.WriteString(result.Stderr)
		if !strings.HasSuffix(result.Stderr, "\n") {
			builder.WriteString("\n")
		}
		builder.WriteString("```\n\n")
	}

	if result.Error != "" {
		builder.WriteString("## Error\n\n")
		builder.WriteString(result.Error)
		builder.WriteString("\n")
	}

	if len(jobExecution.Artifacts) > 0 {
		builder.WriteString("\n## Artifacts\n\n")
		for _, artifact := range jobExecution.Artifacts {
			builder.WriteString(fmt.Sprintf("- `%s` (%d bytes)\n", artifact.RelativePath, artifact.Size))
		}
	} else {
		builder.WriteString("\n## Artifacts\n\n")
		builder.WriteString("No output artifact was detected in the skill workspace.\n")
	}

	// Determine success based on exit code
	success := result.IsSuccess()

	resultData := map[string]interface{}{
		"skill_name":        input.SkillName,
		"script_path":       input.ScriptPath,
		"args":              input.Args,
		"job":               jobExecution.Job,
		"artifacts":         jobExecution.Artifacts,
		"artifact_detected": jobExecution.ArtifactDetected,
		"exit_code":         result.ExitCode,
		"stdout":            result.Stdout,
		"stderr":            result.Stderr,
		"duration_ms":       result.Duration.Milliseconds(),
		"killed":            result.Killed,
	}

	logger.Infof(ctx, "[Tool][ExecuteSkillScript] Script completed with exit code: %d", result.ExitCode)

	return &types.ToolResult{
		Success: success,
		Output:  builder.String(),
		Data:    resultData,
		Error: func() string {
			if !success {
				if result.Error != "" {
					return result.Error
				}
				return fmt.Sprintf("Script exited with code %d", result.ExitCode)
			}
			return ""
		}(),
	}, nil
}

// Cleanup releases any resources
func (t *ExecuteSkillScriptTool) Cleanup(ctx context.Context) error {
	return nil
}

type executeSkillScriptInputLoose struct {
	SkillName  string          `json:"skill_name"`
	ScriptPath string          `json:"script_path"`
	Args       json.RawMessage `json:"args"`
	Input      string          `json:"input"`
}

func parseExecuteSkillScriptInput(raw json.RawMessage) (ExecuteSkillScriptInput, error) {
	var loose executeSkillScriptInputLoose
	if err := json.Unmarshal(raw, &loose); err != nil {
		return ExecuteSkillScriptInput{}, err
	}

	normalizedArgs, err := normalizeExecuteSkillArgs(loose.Args)
	if err != nil {
		return ExecuteSkillScriptInput{}, err
	}

	return ExecuteSkillScriptInput{
		SkillName:  loose.SkillName,
		ScriptPath: loose.ScriptPath,
		Args:       normalizedArgs,
		Input:      loose.Input,
	}, nil
}

func normalizeExecuteSkillArgs(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var args []string
	if err := json.Unmarshal(raw, &args); err == nil {
		return args, nil
	}

	var stringArg string
	if err := json.Unmarshal(raw, &stringArg); err != nil {
		return nil, fmt.Errorf("args must be an array of strings or a JSON-encoded string array")
	}

	stringArg = strings.TrimSpace(stringArg)
	if stringArg == "" {
		return nil, nil
	}

	if strings.HasPrefix(stringArg, "[") {
		if err := json.Unmarshal([]byte(stringArg), &args); err != nil {
			return nil, fmt.Errorf("args string must contain a valid JSON string array: %w", err)
		}
		return args, nil
	}

	return []string{stringArg}, nil
}

type executeSkillInputEnvelope struct {
	Content  string `json:"content"`
	Text     string `json:"text"`
	Markdown string `json:"markdown"`
	Body     string `json:"body"`
	FileName string `json:"file_name"`
	Filename string `json:"filename"`
	Path     string `json:"path"`
}

func normalizeExecuteSkillPayload(args []string, input string) ([]string, string) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return args, input
	}

	var envelope executeSkillInputEnvelope
	if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
		return args, input
	}

	content := firstNonEmptyString(envelope.Content, envelope.Text, envelope.Markdown, envelope.Body)
	if content == "" {
		return args, input
	}

	if firstPositionalArg(args) == "" {
		if candidate := normalizeRelativeMarkdownPath(firstNonEmptyString(envelope.FileName, envelope.Filename, envelope.Path)); candidate != "" {
			args = append([]string{candidate}, args...)
		}
	}

	return args, content
}

func firstPositionalArg(args []string) string {
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "" || strings.HasPrefix(trimmed, "-") {
			continue
		}
		return trimmed
	}
	return ""
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeRelativeMarkdownPath(candidate string) string {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		return ""
	}

	base := filepath.Base(trimmed)
	if base == "." || base == string(filepath.Separator) || strings.HasPrefix(base, ".") {
		return ""
	}

	if ext := strings.ToLower(filepath.Ext(base)); ext == "" {
		base += ".md"
	}

	if strings.HasPrefix(base, "-") {
		return ""
	}

	return base
}
