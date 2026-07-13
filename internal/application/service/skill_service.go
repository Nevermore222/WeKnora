package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Tencent/Xelora/internal/agent/skills"
	"github.com/Tencent/Xelora/internal/logger"
	"github.com/Tencent/Xelora/internal/types/interfaces"
)

// DefaultPreloadedSkillsDir is the default directory for preloaded skills
const DefaultPreloadedSkillsDir = "skills/preloaded"

// skillService implements SkillService interface
type skillService struct {
	loader       *skills.Loader
	preloadedDir string
	mu           sync.RWMutex
	initialized  bool
}

// NewSkillService creates a new skill service
func NewSkillService() interfaces.SkillService {
	// Determine the preloaded skills directory
	preloadedDir := getPreloadedSkillsDir()

	return &skillService{
		preloadedDir: preloadedDir,
		initialized:  false,
	}
}

// getPreloadedSkillsDir returns the path to the preloaded skills directory
func getPreloadedSkillsDir() string {
	// Check if SKILLS_DIR environment variable is set
	if dir := os.Getenv("XELORA_SKILLS_DIR"); dir != "" {
		return dir
	}

	// Try to find the skills directory relative to the executable
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		skillsDir := filepath.Join(execDir, DefaultPreloadedSkillsDir)
		if _, err := os.Stat(skillsDir); err == nil {
			return skillsDir
		}
	}

	// Try current working directory
	cwd, err := os.Getwd()
	if err == nil {
		skillsDir := filepath.Join(cwd, DefaultPreloadedSkillsDir)
		if _, err := os.Stat(skillsDir); err == nil {
			return skillsDir
		}
	}

	// Default to relative path (will be created if needed)
	return DefaultPreloadedSkillsDir
}

// ensureInitialized initializes the loader if not already done
func (s *skillService) ensureInitialized(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	// Check if preloaded directory exists
	if _, err := os.Stat(s.preloadedDir); os.IsNotExist(err) {
		logger.Warnf(ctx, "Preloaded skills directory does not exist: %s", s.preloadedDir)
		// Create the directory to avoid repeated warnings
		if err := os.MkdirAll(s.preloadedDir, 0755); err != nil {
			logger.Warnf(ctx, "Failed to create preloaded skills directory: %v", err)
		}
	}

	// Create loader with preloaded directory
	s.loader = skills.NewLoader([]string{s.preloadedDir})
	s.initialized = true

	logger.Infof(ctx, "Skill service initialized with preloaded directory: %s", s.preloadedDir)

	return nil
}

// ListPreloadedSkills returns metadata for all preloaded skills
func (s *skillService) ListPreloadedSkills(ctx context.Context) ([]*skills.SkillMetadata, error) {
	if err := s.ensureInitialized(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize skill service: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	metadata, err := s.loader.DiscoverSkills()
	if err != nil {
		logger.Errorf(ctx, "Failed to discover preloaded skills: %v", err)
		return nil, fmt.Errorf("failed to discover skills: %w", err)
	}

	logger.Infof(ctx, "Discovered %d preloaded skills", len(metadata))

	return metadata, nil
}

// ListSkillSummaries returns API-safe summaries for all discovered skills.
func (s *skillService) ListSkillSummaries(ctx context.Context) ([]*skills.SkillSummary, error) {
	metadata, err := s.ListPreloadedSkills(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]*skills.SkillSummary, 0, len(metadata))
	for _, meta := range metadata {
		scripts, err := s.scriptSummaries(meta.Name)
		if err != nil {
			logger.Warnf(ctx, "Failed to list scripts for skill %s: %v", meta.Name, err)
		}
		summaries = append(summaries, &skills.SkillSummary{
			Name:        meta.Name,
			Description: meta.Description,
			Source:      "preloaded",
			Status:      "enabled",
			Scripts:     scripts,
		})
	}
	return summaries, nil
}

// GetSkillByName retrieves a skill by its name
func (s *skillService) GetSkillByName(ctx context.Context, name string) (*skills.Skill, error) {
	if err := s.ensureInitialized(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize skill service: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	skill, err := s.loader.LoadSkillInstructions(name)
	if err != nil {
		logger.Errorf(ctx, "Failed to load skill %s: %v", name, err)
		return nil, fmt.Errorf("failed to load skill: %w", err)
	}

	return skill, nil
}

// GetSkillDetail retrieves a loaded skill plus its file and script summaries.
func (s *skillService) GetSkillDetail(ctx context.Context, name string) (*skills.SkillDetail, error) {
	skill, err := s.GetSkillByName(ctx, name)
	if err != nil {
		return nil, err
	}
	files, scripts, err := s.fileAndScriptSummaries(name)
	if err != nil {
		return nil, err
	}
	return &skills.SkillDetail{
		Name:         skill.Name,
		Description:  skill.Description,
		Source:       "preloaded",
		Status:       "enabled",
		Instructions: skill.Instructions,
		Scripts:      scripts,
		Files:        files,
	}, nil
}

// GetSkillFile retrieves an additional file from a skill.
func (s *skillService) GetSkillFile(ctx context.Context, name, path string) (*skills.SkillFile, error) {
	if err := s.ensureInitialized(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize skill service: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	file, err := s.loader.LoadSkillFile(name, path)
	if err != nil {
		return nil, fmt.Errorf("failed to load skill file: %w", err)
	}
	return file, nil
}

// TestRunSkill validates a skill script invocation for Skill Studio. Real
// execution stays behind the workspace-bound tool gateway; this endpoint keeps
// the management API from becoming a second arbitrary script runner.
func (s *skillService) TestRunSkill(ctx context.Context, name string, req skills.SkillTestRunRequest) (*skills.SkillTestRunResult, error) {
	scriptPath := strings.TrimSpace(req.ScriptPath)
	if scriptPath == "" {
		return nil, fmt.Errorf("script_path is required")
	}

	if _, err := s.GetSkillByName(ctx, name); err != nil {
		return nil, err
	}
	file, err := s.GetSkillFile(ctx, name, scriptPath)
	if err != nil {
		return nil, err
	}
	if !file.IsScript {
		return nil, fmt.Errorf("file is not an executable script: %s", scriptPath)
	}

	result := &skills.SkillTestRunResult{
		SkillName:  name,
		ScriptPath: scriptPath,
		Args:       append([]string(nil), req.Args...),
		Success:    false,
		Stdout:     "",
		Stderr:     "",
		Error:      "workspace_required: bind a workspace before running skill test scripts",
		Artifacts:  []skills.SkillTestRunArtifact{},
	}
	if strings.TrimSpace(req.WorkspaceID) != "" {
		result.Error = "execution_unavailable: Skill Studio test-run is validated but not yet wired to workspace execution"
	}
	return result, nil
}

// GetPreloadedDir returns the configured preloaded skills directory
func (s *skillService) GetPreloadedDir() string {
	return s.preloadedDir
}

func (s *skillService) scriptSummaries(name string) ([]skills.SkillScriptSummary, error) {
	_, scripts, err := s.fileAndScriptSummaries(name)
	return scripts, err
}

func (s *skillService) fileAndScriptSummaries(name string) ([]skills.SkillFileSummary, []skills.SkillScriptSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := s.loader.ListSkillFiles(name)
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(files)

	fileSummaries := make([]skills.SkillFileSummary, 0, len(files))
	scriptSummaries := make([]skills.SkillScriptSummary, 0)
	for _, file := range files {
		path := filepath.ToSlash(file)
		isScript := skills.IsScript(path)
		fileSummaries = append(fileSummaries, skills.SkillFileSummary{
			Path:     path,
			IsScript: isScript,
		})
		if isScript {
			scriptSummaries = append(scriptSummaries, skills.SkillScriptSummary{
				Path:     path,
				Language: skills.GetScriptLanguage(path),
			})
		}
	}
	return fileSummaries, scriptSummaries, nil
}
