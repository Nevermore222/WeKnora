package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/Tencent/Xelora/internal/agent/skills"
)

const (
	ProviderStatusAvailable   = "available"
	ProviderStatusUnavailable = "unavailable"
	LocalProviderName         = "local"
)

type ProviderCapability struct {
	Provider                 string    `json:"provider"`
	Status                   string    `json:"status"`
	SupportsSessionWorkspace bool      `json:"supports_session_workspace"`
	SupportsOneOffJob        bool      `json:"supports_one_off_job"`
	SupportsStreamingLogs    bool      `json:"supports_streaming_logs"`
	SupportsFileMount        bool      `json:"supports_file_mount"`
	SupportedRuntimes        []string  `json:"supported_runtimes,omitempty"`
	LastCheckedAt            time.Time `json:"last_checked_at"`
}

type Provider interface {
	Name() string
	ExecuteSkillScript(ctx context.Context, req SkillJobRequest, prepared *skills.PreparedScriptExecution, executor SkillExecutor) (*skills.ScriptExecutionOutcome, error)
	Capability(ctx context.Context) ProviderCapability
}

type LocalProvider struct{}

func NewLocalProvider() *LocalProvider {
	return &LocalProvider{}
}

func (p *LocalProvider) Name() string {
	return LocalProviderName
}

func (p *LocalProvider) ExecuteSkillScript(ctx context.Context, req SkillJobRequest, prepared *skills.PreparedScriptExecution, executor SkillExecutor) (*skills.ScriptExecutionOutcome, error) {
	return executor.ExecutePreparedScript(ctx, prepared)
}

func (p *LocalProvider) Capability(ctx context.Context) ProviderCapability {
	return ProviderCapability{
		Provider:                 p.Name(),
		Status:                   ProviderStatusAvailable,
		SupportsSessionWorkspace: true,
		SupportsOneOffJob:        false,
		SupportsStreamingLogs:    false,
		SupportsFileMount:        true,
		SupportedRuntimes:        []string{"python", "node", "tsx", "bash", "sh"},
		LastCheckedAt:            time.Now(),
	}
}

func selectProvider(providerName string, providers map[string]Provider) (Provider, error) {
	name := providerName
	if name == "" {
		name = LocalProviderName
	}

	provider, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("executor provider not configured: %s", name)
	}
	return provider, nil
}
