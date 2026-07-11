package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Tencent/Xelora/internal/agent/skills"
	"github.com/Tencent/Xelora/internal/sandbox"
)

type OpenSandboxConfig struct {
	BaseURL      string
	APIKey       string
	TemplateID   string
	PythonBinary string
}

func loadOpenSandboxConfigFromEnv() OpenSandboxConfig {
	pythonBinary := strings.TrimSpace(os.Getenv("XELORA_OPENSANDBOX_PYTHON"))
	if pythonBinary == "" {
		pythonBinary = "python3"
	}

	return OpenSandboxConfig{
		BaseURL:      strings.TrimSpace(os.Getenv("XELORA_OPENSANDBOX_BASE_URL")),
		APIKey:       strings.TrimSpace(os.Getenv("XELORA_OPENSANDBOX_API_KEY")),
		TemplateID:   strings.TrimSpace(os.Getenv("XELORA_OPENSANDBOX_TEMPLATE_ID")),
		PythonBinary: pythonBinary,
	}
}

func (c OpenSandboxConfig) Validate() error {
	switch {
	case c.BaseURL == "":
		return fmt.Errorf("XELORA_OPENSANDBOX_BASE_URL is required")
	case c.APIKey == "":
		return fmt.Errorf("XELORA_OPENSANDBOX_API_KEY is required")
	case c.TemplateID == "":
		return fmt.Errorf("XELORA_OPENSANDBOX_TEMPLATE_ID is required")
	case c.PythonBinary == "":
		return fmt.Errorf("OpenSandbox python binary is required")
	default:
		return nil
	}
}

type OpenSandboxProvider struct {
	config OpenSandboxConfig
}

func NewOpenSandboxProvider() *OpenSandboxProvider {
	return &OpenSandboxProvider{
		config: loadOpenSandboxConfigFromEnv(),
	}
}

func (p *OpenSandboxProvider) Name() string {
	return OpenSandboxProviderName
}

func (p *OpenSandboxProvider) Capability(ctx context.Context) ProviderCapability {
	status := ProviderStatusAvailable
	if err := p.config.Validate(); err != nil {
		status = ProviderStatusUnavailable
	}

	return ProviderCapability{
		Provider:                 p.Name(),
		Status:                   status,
		SupportsSessionWorkspace: true,
		SupportsOneOffJob:        false,
		SupportsStreamingLogs:    false,
		SupportsFileMount:        true,
		SupportedRuntimes:        []string{"python", "node", "tsx", "bash", "sh"},
		LastCheckedAt:            time.Now(),
	}
}

type openSandboxRequest struct {
	BasePath   string `json:"base_path"`
	ScriptPath string `json:"script_path"`
	Args       []string `json:"args,omitempty"`
	Stdin      string `json:"stdin,omitempty"`
	TimeoutSec int `json:"timeout_sec"`
}

type openSandboxResponse struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms"`
	SandboxID  string `json:"sandbox_id,omitempty"`
}

func (p *OpenSandboxProvider) ExecuteSkillScript(ctx context.Context, req SkillJobRequest, prepared *skills.PreparedScriptExecution, _ SkillExecutor) (*skills.ScriptExecutionOutcome, error) {
	if err := p.config.Validate(); err != nil {
		return nil, fmt.Errorf("opensandbox provider is not ready: %w", err)
	}

	requestPath, cleanup, err := p.writeRequestFile(prepared)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	cmd := exec.CommandContext(ctx, p.config.PythonBinary, openSandboxHelperPath(), requestPath)
	cmd.Env = append(os.Environ(),
		"XELORA_OPENSANDBOX_BASE_URL="+p.config.BaseURL,
		"XELORA_OPENSANDBOX_API_KEY="+p.config.APIKey,
		"XELORA_OPENSANDBOX_TEMPLATE_ID="+p.config.TemplateID,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("opensandbox helper failed: %w\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	var response openSandboxResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("failed to decode opensandbox helper output: %w\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	return &skills.ScriptExecutionOutcome{
		Result: &sandbox.ExecuteResult{
			Stdout:   response.Stdout,
			Stderr:   response.Stderr,
			ExitCode: response.ExitCode,
			Duration: time.Duration(response.DurationMs) * time.Millisecond,
			Error:    response.Error,
		},
		BasePath:               prepared.BasePath,
		MaterializedInputPaths: prepared.MaterializedInputPaths,
	}, nil
}

func (p *OpenSandboxProvider) writeRequestFile(prepared *skills.PreparedScriptExecution) (string, func(), error) {
	req := openSandboxRequest{
		BasePath:   prepared.BasePath,
		ScriptPath: prepared.ScriptPath,
		Args:       append([]string(nil), prepared.Args...),
		Stdin:      prepared.Stdin,
		TimeoutSec: 60,
	}

	file, err := os.CreateTemp("", "xelora-opensandbox-*.json")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create opensandbox request file: %w", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(&req); err != nil {
		os.Remove(file.Name())
		return "", nil, fmt.Errorf("failed to encode opensandbox request: %w", err)
	}

	return file.Name(), func() {
		_ = os.Remove(file.Name())
	}, nil
}

func openSandboxHelperPath() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("internal", "executor", "scripts", "opensandbox_exec.py")
	}
	return filepath.Join(filepath.Dir(currentFile), "scripts", "opensandbox_exec.py")
}
