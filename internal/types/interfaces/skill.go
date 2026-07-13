package interfaces

import (
	"context"

	"github.com/Tencent/Xelora/internal/agent/skills"
)

// SkillService defines the interface for skill business logic
type SkillService interface {
	// ListPreloadedSkills returns metadata for all preloaded skills
	ListPreloadedSkills(ctx context.Context) ([]*skills.SkillMetadata, error)

	// ListSkillSummaries returns management API summaries for all preloaded skills
	ListSkillSummaries(ctx context.Context) ([]*skills.SkillSummary, error)

	// GetSkillByName retrieves a skill by its name
	GetSkillByName(ctx context.Context, name string) (*skills.Skill, error)

	// GetSkillDetail retrieves a skill with instructions and resource summaries
	GetSkillDetail(ctx context.Context, name string) (*skills.SkillDetail, error)

	// GetSkillFile retrieves an additional file from a skill directory
	GetSkillFile(ctx context.Context, name, path string) (*skills.SkillFile, error)

	// TestRunSkill validates a script invocation for Skill Studio.
	TestRunSkill(ctx context.Context, name string, req skills.SkillTestRunRequest) (*skills.SkillTestRunResult, error)
}
