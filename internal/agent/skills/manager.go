package skills

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Tencent/Xelora/internal/sandbox"
)

type ScriptExecutionOutcome struct {
	Result                 *sandbox.ExecuteResult
	BasePath               string
	MaterializedInputPaths []string
}

// Manager manages skills lifecycle including discovery, loading, and script execution
// It coordinates between the Loader (filesystem operations) and Sandbox (script execution)
type Manager struct {
	loader     *Loader
	sandboxMgr sandbox.Manager

	// Configuration
	skillDirs     []string
	allowedSkills []string // Empty means all skills are allowed
	enabled       bool

	// Cache
	metadataCache []*SkillMetadata
	mu            sync.RWMutex
	installMu     sync.Mutex
}

// ManagerConfig holds configuration for the skill manager
type ManagerConfig struct {
	SkillDirs     []string // Directories to search for skills
	AllowedSkills []string // Skill names whitelist (empty = allow all)
	Enabled       bool     // Whether skills are enabled
}

// NewManager creates a new skill manager with the given configuration
func NewManager(config *ManagerConfig, sandboxMgr sandbox.Manager) *Manager {
	if config == nil {
		config = &ManagerConfig{
			Enabled: false,
		}
	}

	return &Manager{
		loader:        NewLoader(config.SkillDirs),
		sandboxMgr:    sandboxMgr,
		skillDirs:     config.SkillDirs,
		allowedSkills: config.AllowedSkills,
		enabled:       config.Enabled,
	}
}

// IsEnabled returns whether skills are enabled
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// Initialize discovers all skills and caches their metadata
// This should be called at startup
func (m *Manager) Initialize(ctx context.Context) error {
	if !m.enabled {
		return nil
	}

	metadata, err := m.loader.DiscoverSkills()
	if err != nil {
		return fmt.Errorf("failed to discover skills: %w", err)
	}

	// Filter by allowed skills if specified
	if len(m.allowedSkills) > 0 {
		metadata = m.filterAllowedSkills(metadata)
	}

	m.mu.Lock()
	m.metadataCache = metadata
	m.mu.Unlock()

	return nil
}

// filterAllowedSkills filters metadata to only include allowed skills
func (m *Manager) filterAllowedSkills(metadata []*SkillMetadata) []*SkillMetadata {
	if len(m.allowedSkills) == 0 {
		return metadata
	}

	allowedSet := make(map[string]bool)
	for _, name := range m.allowedSkills {
		allowedSet[name] = true
	}

	var filtered []*SkillMetadata
	for _, meta := range metadata {
		if allowedSet[meta.Name] {
			filtered = append(filtered, meta)
		}
	}
	return filtered
}

// GetAllMetadata returns metadata for all discovered skills
// This is used for system prompt injection (Level 1)
func (m *Manager) GetAllMetadata() []*SkillMetadata {
	if !m.enabled {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]*SkillMetadata, len(m.metadataCache))
	copy(result, m.metadataCache)
	return result
}

// LoadSkill loads the full instructions of a skill (Level 2)
func (m *Manager) LoadSkill(ctx context.Context, skillName string) (*Skill, error) {
	if !m.enabled {
		return nil, fmt.Errorf("skills are not enabled")
	}

	// Check if skill is allowed
	if !m.isSkillAllowed(skillName) {
		return nil, fmt.Errorf("skill not allowed: %s", skillName)
	}

	return m.loader.LoadSkillInstructions(skillName)
}

// isSkillAllowed checks if a skill is in the allowed list
func (m *Manager) isSkillAllowed(skillName string) bool {
	if len(m.allowedSkills) == 0 {
		return true
	}
	for _, name := range m.allowedSkills {
		if name == skillName {
			return true
		}
	}
	return false
}

// ReadSkillFile reads an additional file from a skill directory (Level 3)
func (m *Manager) ReadSkillFile(ctx context.Context, skillName, filePath string) (string, error) {
	if !m.enabled {
		return "", fmt.Errorf("skills are not enabled")
	}

	if !m.isSkillAllowed(skillName) {
		return "", fmt.Errorf("skill not allowed: %s", skillName)
	}

	file, err := m.loader.LoadSkillFile(skillName, filePath)
	if err != nil {
		return "", err
	}

	return file.Content, nil
}

// ListSkillFiles lists all files in a skill directory
func (m *Manager) ListSkillFiles(ctx context.Context, skillName string) ([]string, error) {
	if !m.enabled {
		return nil, fmt.Errorf("skills are not enabled")
	}

	if !m.isSkillAllowed(skillName) {
		return nil, fmt.Errorf("skill not allowed: %s", skillName)
	}

	return m.loader.ListSkillFiles(skillName)
}

func (m *Manager) GetSkillBasePath(ctx context.Context, skillName string) (string, error) {
	if !m.enabled {
		return "", fmt.Errorf("skills are not enabled")
	}
	if !m.isSkillAllowed(skillName) {
		return "", fmt.Errorf("skill not allowed: %s", skillName)
	}
	return m.loader.GetSkillBasePath(skillName)
}

// ExecuteScript executes a script from a skill in the sandbox
func (m *Manager) ExecuteScript(ctx context.Context, skillName, scriptPath string, args []string, stdin string) (*sandbox.ExecuteResult, error) {
	outcome, err := m.ExecuteScriptDetailed(ctx, skillName, scriptPath, args, stdin)
	if err != nil {
		return nil, err
	}
	return outcome.Result, nil
}

// ExecuteScriptDetailed executes a script and returns additional runtime metadata
// needed by Xelora-owned gateway layers such as job and artifact tracking.
func (m *Manager) ExecuteScriptDetailed(ctx context.Context, skillName, scriptPath string, args []string, stdin string) (*ScriptExecutionOutcome, error) {
	if !m.enabled {
		return nil, fmt.Errorf("skills are not enabled")
	}

	if !m.isSkillAllowed(skillName) {
		return nil, fmt.Errorf("skill not allowed: %s", skillName)
	}

	// Verify sandbox manager is available
	if m.sandboxMgr == nil {
		return nil, fmt.Errorf("sandbox is not configured")
	}

	// Get the skill base path
	basePath, err := m.loader.GetSkillBasePath(skillName)
	if err != nil {
		return nil, err
	}

	// Load the script file to verify it exists and is a script
	file, err := m.loader.LoadSkillFile(skillName, scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load script: %w", err)
	}

	if !file.IsScript {
		return nil, fmt.Errorf("file is not an executable script: %s", scriptPath)
	}

	if err := m.ensureScriptRuntimeDependencies(ctx, file.Path); err != nil {
		return nil, err
	}

	args, materializedPaths, cleanupInputFile, err := m.materializeScriptInput(basePath, args, stdin)
	if err != nil {
		return nil, err
	}
	if cleanupInputFile != nil {
		defer cleanupInputFile()
	}

	// Prepare execution config
	config := &sandbox.ExecuteConfig{
		Script:  file.Path,
		Args:    args,
		WorkDir: basePath,
		Stdin:   stdin,
	}

	// Execute in sandbox
	result, err := m.sandboxMgr.Execute(ctx, config)
	if err != nil {
		return nil, err
	}

	return &ScriptExecutionOutcome{
		Result:                 result,
		BasePath:               basePath,
		MaterializedInputPaths: materializedPaths,
	}, nil
}

// GetSkillInfo returns detailed information about a skill
func (m *Manager) GetSkillInfo(ctx context.Context, skillName string) (*SkillInfo, error) {
	if !m.enabled {
		return nil, fmt.Errorf("skills are not enabled")
	}

	if !m.isSkillAllowed(skillName) {
		return nil, fmt.Errorf("skill not allowed: %s", skillName)
	}

	skill, err := m.loader.LoadSkillInstructions(skillName)
	if err != nil {
		return nil, err
	}

	files, err := m.loader.ListSkillFiles(skillName)
	if err != nil {
		files = []string{} // Non-fatal error
	}

	return &SkillInfo{
		Name:         skill.Name,
		Description:  skill.Description,
		BasePath:     skill.BasePath,
		Instructions: skill.Instructions,
		Files:        files,
	}, nil
}

// SkillInfo provides detailed information about a skill
type SkillInfo struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	BasePath     string   `json:"base_path"`
	Instructions string   `json:"instructions"`
	Files        []string `json:"files"`
}

// Reload refreshes the skill cache by rediscovering all skills
func (m *Manager) Reload(ctx context.Context) error {
	if !m.enabled {
		return nil
	}

	metadata, err := m.loader.Reload()
	if err != nil {
		return err
	}

	if len(m.allowedSkills) > 0 {
		metadata = m.filterAllowedSkills(metadata)
	}

	m.mu.Lock()
	m.metadataCache = metadata
	m.mu.Unlock()

	return nil
}

// Cleanup releases resources
func (m *Manager) Cleanup(ctx context.Context) error {
	if m.sandboxMgr != nil {
		return m.sandboxMgr.Cleanup(ctx)
	}
	return nil
}

func (m *Manager) ensureScriptRuntimeDependencies(ctx context.Context, scriptPath string) error {
	ext := filepath.Ext(scriptPath)
	if ext != ".ts" && ext != ".js" {
		return nil
	}

	scriptDir := filepath.Dir(scriptPath)
	packageJSON := filepath.Join(scriptDir, "package.json")
	if _, err := os.Stat(packageJSON); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to inspect package.json: %w", err)
	}

	nodeModules := filepath.Join(scriptDir, "node_modules")
	if info, err := os.Stat(nodeModules); err == nil && info.IsDir() {
		return nil
	}

	m.installMu.Lock()
	defer m.installMu.Unlock()

	if info, err := os.Stat(nodeModules); err == nil && info.IsDir() {
		return nil
	}

	lockFile := filepath.Join(scriptDir, "package-lock.json")
	installArgs := []string{"install", "--omit=dev"}
	if _, err := os.Stat(lockFile); err == nil {
		installArgs = []string{"ci", "--omit=dev"}
	}

	cmd := exec.CommandContext(ctx, "npm", installArgs...)
	cmd.Dir = scriptDir
	cmd.Env = append(os.Environ(),
		"NODE_ENV=production",
		"NPM_CONFIG_FUND=false",
		"NPM_CONFIG_AUDIT=false",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install skill runtime dependencies in %s: %w\n%s", scriptDir, err, string(output))
	}

	return nil
}

func (m *Manager) materializeScriptInput(basePath string, args []string, stdin string) ([]string, []string, func(), error) {
	if strings.TrimSpace(stdin) == "" {
		return args, nil, nil, nil
	}

	if idx := firstPositionalArgIndex(args); idx >= 0 {
		targetPath, err := resolveSkillRelativePath(basePath, args[idx])
		if err != nil {
			return nil, nil, nil, err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to prepare input file directory: %w", err)
		}
		if err := os.WriteFile(targetPath, []byte(stdin), 0o644); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to materialize skill input file: %w", err)
		}
		return args, []string{targetPath}, nil, nil
	}

	fileName := buildGeneratedMarkdownName()
	targetPath, err := resolveSkillRelativePath(basePath, fileName)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := os.WriteFile(targetPath, []byte(stdin), 0o644); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to materialize generated markdown file: %w", err)
	}

	return append([]string{fileName}, args...), []string{targetPath}, nil, nil
}

func firstPositionalArgIndex(args []string) int {
	for i, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "" || strings.HasPrefix(trimmed, "-") {
			continue
		}
		return i
	}
	return -1
}

func resolveSkillRelativePath(basePath, candidate string) (string, error) {
	cleaned := filepath.Clean(strings.TrimSpace(candidate))
	if cleaned == "." || cleaned == "" || filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "..") {
		return "", fmt.Errorf("invalid skill file path: %s", candidate)
	}
	return filepath.Join(basePath, cleaned), nil
}

func buildGeneratedMarkdownName() string {
	return "generated-" + time.Now().Format("20060102-150405") + "-" + strconv.FormatInt(time.Now().UnixNano()%1_000_000, 10) + ".md"
}
