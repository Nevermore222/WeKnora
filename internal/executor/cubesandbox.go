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

const CubeSandboxProviderName = "cubesandbox"

type CubeSandboxConfig struct {
	APIURL       string
	APIKey       string
	TemplateID   string
	SSLCertFile  string
	PythonBinary string
}

func loadCubeSandboxConfigFromEnv() CubeSandboxConfig {
	pythonBinary := strings.TrimSpace(os.Getenv("XELORA_CUBESANDBOX_PYTHON"))
	if pythonBinary == "" {
		pythonBinary = "python3"
	}

	return CubeSandboxConfig{
		APIURL:       strings.TrimSpace(os.Getenv("E2B_API_URL")),
		APIKey:       strings.TrimSpace(os.Getenv("E2B_API_KEY")),
		TemplateID:   strings.TrimSpace(os.Getenv("CUBE_TEMPLATE_ID")),
		SSLCertFile:  strings.TrimSpace(os.Getenv("SSL_CERT_FILE")),
		PythonBinary: pythonBinary,
	}
}

func (c CubeSandboxConfig) Validate() error {
	switch {
	case c.APIURL == "":
		return fmt.Errorf("E2B_API_URL is required")
	case c.APIKey == "":
		return fmt.Errorf("E2B_API_KEY is required")
	case c.TemplateID == "":
		return fmt.Errorf("CUBE_TEMPLATE_ID is required")
	case c.PythonBinary == "":
		return fmt.Errorf("CubeSandbox python binary is required")
	default:
		return nil
	}
}

type CubeSandboxProvider struct {
	config CubeSandboxConfig
}

func NewCubeSandboxProvider() *CubeSandboxProvider {
	return &CubeSandboxProvider{
		config: loadCubeSandboxConfigFromEnv(),
	}
}

func (p *CubeSandboxProvider) Name() string {
	return CubeSandboxProviderName
}

func (p *CubeSandboxProvider) Capability(ctx context.Context) ProviderCapability {
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

type cubeSandboxRequest struct {
	BasePath   string            `json:"base_path"`
	ScriptPath string            `json:"script_path"`
	Args       []string          `json:"args,omitempty"`
	Stdin      string            `json:"stdin,omitempty"`
	TimeoutSec int               `json:"timeout_sec"`
	Env        map[string]string `json:"env,omitempty"`
}

type cubeSandboxResponse struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms"`
	SandboxID  string `json:"sandbox_id,omitempty"`
}

func (p *CubeSandboxProvider) ExecuteSkillScript(ctx context.Context, req SkillJobRequest, prepared *skills.PreparedScriptExecution, _ SkillExecutor) (*skills.ScriptExecutionOutcome, error) {
	if err := p.config.Validate(); err != nil {
		return nil, fmt.Errorf("cubesandbox provider is not ready: %w", err)
	}

	requestPath, cleanup, err := p.writeRequestFile(prepared)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	cmd := exec.CommandContext(ctx, p.config.PythonBinary, cubeSandboxHelperPath(), requestPath)
	cmd.Env = append(os.Environ(),
		"E2B_API_URL="+p.config.APIURL,
		"E2B_API_KEY="+p.config.APIKey,
		"CUBE_TEMPLATE_ID="+p.config.TemplateID,
	)
	if p.config.SSLCertFile != "" {
		cmd.Env = append(cmd.Env, "SSL_CERT_FILE="+p.config.SSLCertFile)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cubesandbox helper failed: %w\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	var response cubeSandboxResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("failed to decode cubesandbox helper output: %w\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
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

func (p *CubeSandboxProvider) writeRequestFile(prepared *skills.PreparedScriptExecution) (string, func(), error) {
	req := cubeSandboxRequest{
		BasePath:   prepared.BasePath,
		ScriptPath: prepared.ScriptPath,
		Args:       append([]string(nil), prepared.Args...),
		Stdin:      prepared.Stdin,
		TimeoutSec: 60,
	}

	file, err := os.CreateTemp("", "xelora-cubesandbox-*.json")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create cubesandbox request file: %w", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(&req); err != nil {
		os.Remove(file.Name())
		return "", nil, fmt.Errorf("failed to encode cubesandbox request: %w", err)
	}

	return file.Name(), func() {
		_ = os.Remove(file.Name())
	}, nil
}

func cubeSandboxHelperPath() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("internal", "executor", "scripts", "cubesandbox_exec.py")
	}
	return filepath.Join(filepath.Dir(currentFile), "scripts", "cubesandbox_exec.py")
}
