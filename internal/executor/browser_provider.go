package executor

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Tencent/Xelora/internal/agent/skills"
	"github.com/Tencent/Xelora/internal/sandbox"
)

// BrowserProviderConfig holds the configuration for the controlled Docker
// browser provider. It reuses the sandbox Docker execution to run the
// browser-snapshot skill script, so the browser binary runs inside the
// sandbox image rather than on the Xelora host.
type BrowserProviderConfig struct {
	Image       string
	Timeout     time.Duration
	VolumesFrom string
	SkillName   string
	ScriptPath  string
}

func loadBrowserProviderConfigFromEnv() BrowserProviderConfig {
	image := os.Getenv("XELORA_BROWSER_IMAGE")
	if image == "" {
		image = sandbox.DefaultDockerImage
	}

	timeout := sandbox.DefaultTimeout
	if raw := os.Getenv("XELORA_BROWSER_TIMEOUT"); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	volumesFrom := os.Getenv("XELORA_SANDBOX_DOCKER_VOLUMES_FROM")

	return BrowserProviderConfig{
		Image:       image,
		Timeout:     timeout,
		VolumesFrom: volumesFrom,
		SkillName:   "browser-snapshot",
		ScriptPath:  "scripts/browser_snapshot.py",
	}
}

// ControlledDockerBrowserProvider runs browser navigation tasks by executing
// the browser-snapshot skill script inside the controlled Docker sandbox.
// The provider owns browser launch and page capture mechanics; the gateway
// owns job identity, artifact registration, and workspace routing.
type ControlledDockerBrowserProvider struct {
	config BrowserProviderConfig
}

func NewControlledDockerBrowserProvider() *ControlledDockerBrowserProvider {
	return &ControlledDockerBrowserProvider{
		config: loadBrowserProviderConfigFromEnv(),
	}
}

func (p *ControlledDockerBrowserProvider) Name() string {
	return ControlledDockerBrowserProviderName
}

func (p *ControlledDockerBrowserProvider) Capability(ctx context.Context) ProviderCapability {
	cfg := sandbox.DefaultConfig()
	cfg.Type = sandbox.SandboxTypeDocker
	cfg.FallbackEnabled = false
	cfg.DockerImage = p.config.Image
	cfg.DockerVolumesFrom = p.config.VolumesFrom
	cfg.DefaultTimeout = p.config.Timeout

	status := ProviderStatusUnavailable
	if sandbox.NewDockerSandbox(cfg).IsAvailable(ctx) {
		status = ProviderStatusAvailable
	}

	return ProviderCapability{
		Provider:                 p.Name(),
		Status:                   status,
		SupportsSessionWorkspace: true,
		SupportsOneOffJob:        false,
		SupportsStreamingLogs:    false,
		SupportsFileMount:        true,
		SupportedRuntimes:        []string{"python", "bash", "sh"},
		LastCheckedAt:            time.Now(),
	}
}

func (p *ControlledDockerBrowserProvider) ExecuteBrowserTask(ctx context.Context, req BrowserJobRequest, outputDir string, skillExecutor SkillExecutor) (*BrowserTaskResult, error) {
	if skillExecutor == nil {
		return nil, fmt.Errorf("skill executor is required for browser task execution")
	}

	captureMode := string(req.CaptureMode)
	if captureMode == "" {
		captureMode = string(BrowserCaptureScreenshot)
	}

	args := []string{req.URL, captureMode}

	prepared, err := skillExecutor.PrepareScriptExecution(ctx, p.config.SkillName, p.config.ScriptPath, args, req.Input)
	if err != nil {
		return nil, fmt.Errorf("prepare browser snapshot script: %w", err)
	}
	if prepared.Cleanup != nil {
		defer prepared.Cleanup()
	}

	// Run the script with the workspace output directory as the working
	// directory so screenshot and content files land there.
	if outputDir != "" {
		prepared.WorkDir = outputDir
	}

	timeout := p.config.Timeout
	if req.Timeout > 0 {
		timeout = req.Timeout
	}

	manager, err := sandbox.NewManager(&sandbox.Config{
		Type:              sandbox.SandboxTypeDocker,
		FallbackEnabled:   false,
		DefaultTimeout:    timeout,
		DockerImage:       p.config.Image,
		DockerVolumesFrom: p.config.VolumesFrom,
		MaxMemory:         sandbox.DefaultMemoryLimit,
		MaxCPU:            sandbox.DefaultCPULimit,
	})
	if err != nil {
		return nil, fmt.Errorf("controlled docker browser provider is not ready: %w", err)
	}

	result, err := manager.Execute(ctx, &sandbox.ExecuteConfig{
		Script:  prepared.ScriptPath,
		Args:    prepared.Args,
		WorkDir: preparedWorkDir(prepared),
		Stdin:   prepared.Stdin,
		Timeout: timeout,
	})
	if err != nil {
		return nil, err
	}

	return &BrowserTaskResult{
		Result: result,
	}, nil
}

// browserProviderExecutor is a minimal adapter that satisfies the SkillExecutor
// interface when a browser provider is invoked without an explicit skill
// executor. This is used for capability checks.
type nopSkillExecutor struct{}

func (nopSkillExecutor) GetSkillBasePath(ctx context.Context, skillName string) (string, error) {
	return "", nil
}
func (nopSkillExecutor) PrepareScriptExecution(ctx context.Context, skillName, scriptPath string, args []string, stdin string) (*skills.PreparedScriptExecution, error) {
	return &skills.PreparedScriptExecution{}, nil
}
func (nopSkillExecutor) ExecutePreparedScript(ctx context.Context, prepared *skills.PreparedScriptExecution) (*skills.ScriptExecutionOutcome, error) {
	return &skills.ScriptExecutionOutcome{Result: &sandbox.ExecuteResult{}}, nil
}
func (nopSkillExecutor) ExecuteScriptDetailed(ctx context.Context, skillName, scriptPath string, args []string, stdin string) (*skills.ScriptExecutionOutcome, error) {
	return &skills.ScriptExecutionOutcome{Result: &sandbox.ExecuteResult{}}, nil
}
