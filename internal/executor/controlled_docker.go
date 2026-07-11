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

type ControlledDockerConfig struct {
	Image       string
	Timeout     time.Duration
	VolumesFrom string
}

func loadControlledDockerConfigFromEnv() ControlledDockerConfig {
	image := os.Getenv("XELORA_SANDBOX_DOCKER_IMAGE")
	if image == "" {
		image = sandbox.DefaultDockerImage
	}

	timeout := sandbox.DefaultTimeout
	if raw := os.Getenv("XELORA_SANDBOX_TIMEOUT"); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	return ControlledDockerConfig{
		Image:       image,
		Timeout:     timeout,
		VolumesFrom: os.Getenv("XELORA_SANDBOX_DOCKER_VOLUMES_FROM"),
	}
}

type ControlledDockerProvider struct {
	config ControlledDockerConfig
}

func NewControlledDockerProvider() *ControlledDockerProvider {
	return &ControlledDockerProvider{
		config: loadControlledDockerConfigFromEnv(),
	}
}

func (p *ControlledDockerProvider) Name() string {
	return ControlledDockerProviderName
}

func (p *ControlledDockerProvider) Capability(ctx context.Context) ProviderCapability {
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
		SupportedRuntimes:        []string{"python", "node", "tsx", "bash", "sh"},
		LastCheckedAt:            time.Now(),
	}
}

func (p *ControlledDockerProvider) ExecuteSkillScript(ctx context.Context, req SkillJobRequest, prepared *skills.PreparedScriptExecution, _ SkillExecutor) (*skills.ScriptExecutionOutcome, error) {
	if prepared == nil {
		return nil, fmt.Errorf("prepared script execution is required")
	}

	manager, err := sandbox.NewManager(&sandbox.Config{
		Type:              sandbox.SandboxTypeDocker,
		FallbackEnabled:   false,
		DefaultTimeout:    p.config.Timeout,
		DockerImage:       p.config.Image,
		DockerVolumesFrom: p.config.VolumesFrom,
		MaxMemory:         sandbox.DefaultMemoryLimit,
		MaxCPU:            sandbox.DefaultCPULimit,
	})
	if err != nil {
		return nil, fmt.Errorf("controlled docker provider is not ready: %w", err)
	}

	result, err := manager.Execute(ctx, &sandbox.ExecuteConfig{
		Script:  prepared.ScriptPath,
		Args:    prepared.Args,
		WorkDir: preparedWorkDir(prepared),
		Stdin:   prepared.Stdin,
		Timeout: p.config.Timeout,
	})
	if err != nil {
		return nil, err
	}

	return &skills.ScriptExecutionOutcome{
		Result:                 result,
		BasePath:               prepared.BasePath,
		WorkDir:                preparedWorkDir(prepared),
		MaterializedInputPaths: prepared.MaterializedInputPaths,
	}, nil
}

// preparedWorkDir returns the effective working directory from a prepared
// script execution, falling back to BasePath when WorkDir is not set.
func preparedWorkDir(prepared *skills.PreparedScriptExecution) string {
	if prepared != nil && prepared.WorkDir != "" {
		return prepared.WorkDir
	}
	if prepared != nil {
		return prepared.BasePath
	}
	return ""
}
