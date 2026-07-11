package executor

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Tencent/Xelora/internal/agent/skills"
	"github.com/Tencent/Xelora/internal/sandbox"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/google/uuid"
)

type SkillExecutor interface {
	GetSkillBasePath(ctx context.Context, skillName string) (string, error)
	PrepareScriptExecution(ctx context.Context, skillName, scriptPath string, args []string, stdin string) (*skills.PreparedScriptExecution, error)
	ExecutePreparedScript(ctx context.Context, prepared *skills.PreparedScriptExecution) (*skills.ScriptExecutionOutcome, error)
	ExecuteScriptDetailed(ctx context.Context, skillName, scriptPath string, args []string, stdin string) (*skills.ScriptExecutionOutcome, error)
}

type SkillJobExecution struct {
	Job              *SkillJob
	Workspace        *WorkspaceRecord
	Provider         ProviderCapability
	Artifacts        []*ArtifactRecord
	ArtifactDetected bool
	Result           *sandbox.ExecuteResult
}

type Gateway struct {
	providers        map[string]Provider
	browserProviders map[string]BrowserProvider
}

func NewGateway() *Gateway {
	return NewGatewayWithProviders(
		NewControlledDockerProvider(),
		NewLocalProvider(),
		NewOpenSandboxProvider(),
		NewCubeSandboxProvider(),
	)
}

func NewGatewayWithProviders(providers ...Provider) *Gateway {
	registry := make(map[string]Provider, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		registry[provider.Name()] = provider
	}
	browserRegistry := make(map[string]BrowserProvider)
	gateway := &Gateway{providers: registry, browserProviders: browserRegistry}
	gateway.RegisterBrowserProvider(NewControlledDockerBrowserProvider())
	return gateway
}

// RegisterBrowserProvider adds a browser provider to the gateway's registry.
func (g *Gateway) RegisterBrowserProvider(provider BrowserProvider) {
	if provider == nil || g.browserProviders == nil {
		g.browserProviders = make(map[string]BrowserProvider)
	}
	if provider != nil {
		g.browserProviders[provider.Name()] = provider
	}
}

// RunBrowserTaskJob dispatches a browser navigation task through the gateway.
// It mirrors RunSkillScriptJob: resolve the conversation output context,
// enforce boundary checks, select the browser provider, execute the task,
// and detect/register artifacts via the same file-snapshot diffing.
func (g *Gateway) RunBrowserTaskJob(ctx context.Context, req BrowserJobRequest, executor SkillExecutor) (*BrowserTaskExecution, error) {
	provider, err := selectBrowserProvider(req.Provider, g.browserProviders)
	if err != nil {
		return nil, err
	}

	outputCtx, err := requireWorkspaceOutputContext(req.SessionID, req.WorkspaceBinding)
	if err != nil {
		return nil, err
	}
	artifactBase := outputCtx.EffectiveRootDir
	if err := validateWorkspaceRoot(artifactBase); err != nil {
		return nil, err
	}
	if !outputCtx.IsWithinWorkspaceRoot(artifactBase) {
		return nil, fmt.Errorf("workspace_path_escape: browser output root escapes bound workspace boundary")
	}
	providerCapability := provider.Capability(ctx)

	now := time.Now()
	workspace := &WorkspaceRecord{
		ID:         outputCtx.WorkspaceID,
		SessionID:  req.SessionID,
		SkillName:  "browser-snapshot",
		RootPath:   artifactBase,
		WorkingDir: ".",
		Provider:   provider.Name(),
		Status:     WorkspaceStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: now,
	}
	workspace.RootPath = outputCtx.EffectiveRootDir
	workspace.WorkingDir = outputCtx.EffectiveRootDir

	preSnapshot, err := snapshotFiles(artifactBase)
	if err != nil {
		return nil, err
	}

	job := &BrowserJob{
		ID:                 uuid.NewString(),
		WorkspaceID:        workspace.ID,
		SessionID:          req.SessionID,
		AssistantMessageID: req.AssistantMessageID,
		RequestID:          req.RequestID,
		ToolCallID:         req.ToolCallID,
		URL:                req.URL,
		CaptureMode:        req.CaptureMode,
		Provider:           provider.Name(),
		WorkingDir:         workspace.WorkingDir,
		Status:             JobStatusRunning,
		StartedAt:          now,
	}
	if job.CaptureMode == "" {
		job.CaptureMode = BrowserCaptureScreenshot
	}

	taskResult, err := provider.ExecuteBrowserTask(ctx, req, artifactBase, executor)
	if err != nil {
		return nil, err
	}

	postSnapshot, snapshotErr := snapshotFiles(artifactBase)
	if snapshotErr == nil {
		artifacts := detectArtifacts(artifactBase, workspace.ID, job.ID, req.SessionID, preSnapshot, postSnapshot, nil)
		job.ArtifactCount = len(artifacts)
		job.FinishedAt = job.StartedAt.Add(taskResult.Result.Duration)
		job.DurationMs = taskResult.Result.Duration.Milliseconds()
		job.ExitCode = taskResult.Result.ExitCode
		job.StdoutSummary = summarizeOutput(taskResult.Result.Stdout)
		job.StderrSummary = summarizeOutput(taskResult.Result.Stderr)
		if taskResult.Result.Error != "" {
			job.Error = taskResult.Result.Error
		}
		if taskResult.Result.IsSuccess() {
			job.Status = JobStatusSucceeded
		} else {
			job.Status = JobStatusFailed
			if job.Error == "" {
				job.Error = "browser task failed"
			}
		}
		return &BrowserTaskExecution{
			Job:              job,
			Workspace:        workspace,
			Provider:         providerCapability,
			Artifacts:        artifacts,
			ArtifactDetected: len(artifacts) > 0,
			Result:           taskResult.Result,
		}, nil
	}

	job.FinishedAt = job.StartedAt.Add(taskResult.Result.Duration)
	job.DurationMs = taskResult.Result.Duration.Milliseconds()
	job.ExitCode = taskResult.Result.ExitCode
	job.StdoutSummary = summarizeOutput(taskResult.Result.Stdout)
	job.StderrSummary = summarizeOutput(taskResult.Result.Stderr)
	if taskResult.Result.Error != "" {
		job.Error = taskResult.Result.Error
	}
	if taskResult.Result.IsSuccess() {
		job.Status = JobStatusSucceeded
	} else {
		job.Status = JobStatusFailed
		if job.Error == "" {
			job.Error = "browser task failed"
		}
	}

	return &BrowserTaskExecution{
		Job:              job,
		Workspace:        workspace,
		Provider:         providerCapability,
		Artifacts:        nil,
		ArtifactDetected: false,
		Result:           taskResult.Result,
	}, nil
}

func buildBrowserWorkspaceID(sessionID string) string {
	if sessionID != "" {
		return "session-" + sanitizeWorkspaceIDPart(sessionID) + "-browser"
	}
	return "browser-snapshot"
}

func selectBrowserProvider(providerName string, providers map[string]BrowserProvider) (BrowserProvider, error) {
	name := providerName
	if name == "" {
		name = ControlledDockerBrowserProviderName
	}
	provider, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("browser provider not configured: %s", name)
	}
	return provider, nil
}

func (g *Gateway) RunSkillScriptJob(ctx context.Context, req SkillJobRequest, executor SkillExecutor) (*SkillJobExecution, error) {
	provider, err := selectProvider(req.Provider, g.providers)
	if err != nil {
		return nil, err
	}
	outputCtx, err := requireWorkspaceOutputContext(req.SessionID, req.WorkspaceBinding)
	if err != nil {
		return nil, err
	}
	artifactBase := outputCtx.EffectiveRootDir
	if err := validateWorkspaceRoot(artifactBase); err != nil {
		return nil, err
	}
	if !outputCtx.IsWithinWorkspaceRoot(artifactBase) {
		return nil, fmt.Errorf("workspace_path_escape: output root escapes bound workspace boundary")
	}
	// Provider selection happens inside the gateway so Xelora retains ownership
	// of workspace IDs, policy checks, job state, and artifact registration
	// regardless of which sandbox backend is active.
	providerCapability := provider.Capability(ctx)

	now := time.Now()
	workspace := buildWorkspaceRecord(req, artifactBase, provider.Name(), now)
	workspace.ID = outputCtx.WorkspaceID
	workspace.RootPath = outputCtx.EffectiveRootDir
	workspace.WorkingDir = outputCtx.EffectiveRootDir

	preSnapshot, err := snapshotFiles(artifactBase)
	if err != nil {
		return nil, err
	}

	jobID := uuid.NewString()
	job := &SkillJob{
		ID:                 jobID,
		WorkspaceID:        workspace.ID,
		SessionID:          req.SessionID,
		AssistantMessageID: req.AssistantMessageID,
		RequestID:          req.RequestID,
		ToolCallID:         req.ToolCallID,
		SkillName:          req.SkillName,
		ScriptPath:         req.ScriptPath,
		Args:               append([]string(nil), req.Args...),
		Provider:           provider.Name(),
		WorkingDir:         workspace.WorkingDir,
		Status:             JobStatusRunning,
		StartedAt:          now,
	}

	prepared, err := executor.PrepareScriptExecution(ctx, req.SkillName, req.ScriptPath, req.Args, req.Input)
	if err != nil {
		return nil, err
	}
	if prepared.Cleanup != nil {
		defer prepared.Cleanup()
	}

	stagingCleanup, err := stagePreparedInputs(prepared, artifactBase, jobID)
	if err != nil {
		return nil, err
	}
	if stagingCleanup != nil {
		defer stagingCleanup()
	}
	prepared.WorkDir = outputCtx.EffectiveRootDir

	outcome, err := provider.ExecuteSkillScript(ctx, req, prepared, executor)
	if err != nil {
		return nil, err
	}

	postSnapshot, snapshotErr := snapshotFiles(artifactBase)
	if snapshotErr == nil {
		artifacts := detectArtifacts(artifactBase, workspace.ID, job.ID, req.SessionID, preSnapshot, postSnapshot, outcome.MaterializedInputPaths)
		job.ArtifactCount = len(artifacts)
		job.FinishedAt = job.StartedAt.Add(outcome.Result.Duration)
		job.DurationMs = outcome.Result.Duration.Milliseconds()
		job.ExitCode = outcome.Result.ExitCode
		job.StdoutSummary = summarizeOutput(outcome.Result.Stdout)
		job.StderrSummary = summarizeOutput(outcome.Result.Stderr)
		if outcome.Result.Error != "" {
			job.Error = outcome.Result.Error
		}
		if outcome.Result.IsSuccess() {
			job.Status = JobStatusSucceeded
		} else {
			job.Status = JobStatusFailed
			if job.Error == "" {
				job.Error = "script execution failed"
			}
		}
		return &SkillJobExecution{
			Job:              job,
			Workspace:        workspace,
			Provider:         providerCapability,
			Artifacts:        artifacts,
			ArtifactDetected: len(artifacts) > 0,
			Result:           outcome.Result,
		}, nil
	}

	job.FinishedAt = job.StartedAt.Add(outcome.Result.Duration)
	job.DurationMs = outcome.Result.Duration.Milliseconds()
	job.ExitCode = outcome.Result.ExitCode
	job.StdoutSummary = summarizeOutput(outcome.Result.Stdout)
	job.StderrSummary = summarizeOutput(outcome.Result.Stderr)
	if outcome.Result.Error != "" {
		job.Error = outcome.Result.Error
	}
	if outcome.Result.IsSuccess() {
		job.Status = JobStatusSucceeded
	} else {
		job.Status = JobStatusFailed
		if job.Error == "" {
			job.Error = "script execution failed"
		}
	}

	return &SkillJobExecution{
		Job:              job,
		Workspace:        workspace,
		Provider:         providerCapability,
		Artifacts:        nil,
		ArtifactDetected: false,
		Result:           outcome.Result,
	}, nil
}

type fileSnapshot struct {
	RelativePath string
	AbsolutePath string
	Size         int64
	ModifiedAt   time.Time
}

func snapshotFiles(basePath string) (map[string]fileSnapshot, error) {
	snapshots := make(map[string]fileSnapshot)
	err := filepath.WalkDir(basePath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if shouldIgnoreDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldIgnoreFile(path) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(basePath, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		snapshots[rel] = fileSnapshot{
			RelativePath: rel,
			AbsolutePath: path,
			Size:         info.Size(),
			ModifiedAt:   info.ModTime(),
		}
		return nil
	})
	return snapshots, err
}

func detectArtifacts(basePath, workspaceID, jobID, sessionID string, pre, post map[string]fileSnapshot, materializedInputPaths []string) []*ArtifactRecord {
	skipInputs := make(map[string]struct{}, len(materializedInputPaths))
	for _, path := range materializedInputPaths {
		rel, err := filepath.Rel(basePath, path)
		if err != nil {
			continue
		}
		skipInputs[filepath.ToSlash(rel)] = struct{}{}
	}

	artifacts := make([]*ArtifactRecord, 0)
	for rel, after := range post {
		if _, skip := skipInputs[rel]; skip {
			continue
		}
		if !isArtifactFile(rel) {
			continue
		}
		before, exists := pre[rel]
		if exists && before.Size == after.Size && before.ModifiedAt.Equal(after.ModifiedAt) {
			continue
		}
		artifacts = append(artifacts, &ArtifactRecord{
			ID:           uuid.NewString(),
			WorkspaceID:  workspaceID,
			JobID:        jobID,
			SessionID:    sessionID,
			Name:         filepath.Base(rel),
			RelativePath: rel,
			AbsolutePath: after.AbsolutePath,
			ContentType:  mime.TypeByExtension(strings.ToLower(filepath.Ext(rel))),
			Kind:         detectArtifactKind(rel),
			PreviewState: detectArtifactPreviewState(rel),
			ChangeType:   detectArtifactChangeType(exists),
			Size:         after.Size,
			ModifiedAt:   after.ModifiedAt,
		})
	}

	slices.SortFunc(artifacts, func(a, b *ArtifactRecord) int {
		return strings.Compare(a.RelativePath, b.RelativePath)
	})

	return artifacts
}

func buildWorkspaceRecord(req SkillJobRequest, rootPath, providerName string, now time.Time) *WorkspaceRecord {
	return &WorkspaceRecord{
		ID:         buildWorkspaceID(req.SessionID, req.SkillName),
		SessionID:  req.SessionID,
		SkillName:  req.SkillName,
		RootPath:   rootPath,
		WorkingDir: ".",
		Provider:   providerName,
		Status:     WorkspaceStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: now,
	}
}

func buildWorkspaceID(sessionID, skillName string) string {
	skillPart := sanitizeWorkspaceIDPart(skillName)
	if sessionID != "" {
		return "session-" + sanitizeWorkspaceIDPart(sessionID) + "-skill-" + skillPart
	}
	return "skill-" + skillPart
}

func sanitizeWorkspaceIDPart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "default"
	}

	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		case r == '.', r == '_', r == '-':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}

	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "default"
	}
	return result
}

func shouldIgnoreDir(name string) bool {
	switch name {
	case "node_modules", "__pycache__", ".git", ".xelora":
		return true
	default:
		return false
	}
}

func requireWorkspaceOutputContext(sessionID string, binding *types.SessionWorkspaceBinding) (ConversationOutputContext, error) {
	outputCtx := ResolveConversationOutputContext(sessionID, binding)
	if !outputCtx.WriteAllowed || outputCtx.EffectiveRootDir == "" {
		code := string(outputCtx.FailureCode)
		if code == "" || outputCtx.Mode == ConversationOutputModeUnbound {
			code = string(ConversationOutputFailureWorkspaceRequired)
		}
		message := strings.TrimSpace(outputCtx.FailureMessage)
		if message == "" {
			message = "bind a workspace before running file tools"
		}
		return outputCtx, fmt.Errorf("%s: %s", code, message)
	}
	return outputCtx, nil
}

func validateWorkspaceRoot(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("workspace_not_found: bound workspace no longer exists")
		}
		return fmt.Errorf("workspace_access_denied: inspect workspace: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("workspace_not_found: bound workspace is not a directory")
	}
	return nil
}

func stagePreparedInputs(prepared *skills.PreparedScriptExecution, workspaceRoot, jobID string) (func(), error) {
	if prepared == nil || len(prepared.MaterializedInputPaths) == 0 {
		return nil, nil
	}
	stageDir := filepath.Join(workspaceRoot, ".xelora", "jobs", jobID)
	if err := os.MkdirAll(stageDir, 0o700); err != nil {
		return nil, fmt.Errorf("stage skill input: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(stageDir) }
	replacements := make(map[string]string, len(prepared.MaterializedInputPaths)*3)
	stagedPaths := make([]string, 0, len(prepared.MaterializedInputPaths))
	for _, source := range prepared.MaterializedInputPaths {
		target := filepath.Join(stageDir, filepath.Base(source))
		if err := copyFile(source, target); err != nil {
			cleanup()
			return nil, err
		}
		relTarget, err := filepath.Rel(workspaceRoot, target)
		if err != nil {
			cleanup()
			return nil, err
		}
		replacements[filepath.Clean(source)] = relTarget
		replacements[filepath.Base(source)] = relTarget
		if prepared.BasePath != "" {
			if relSource, relErr := filepath.Rel(prepared.BasePath, source); relErr == nil {
				replacements[filepath.Clean(relSource)] = relTarget
			}
		}
		stagedPaths = append(stagedPaths, target)
	}
	for index, arg := range prepared.Args {
		if replacement, ok := replacements[filepath.Clean(arg)]; ok {
			prepared.Args[index] = replacement
		}
	}
	prepared.MaterializedInputPaths = stagedPaths
	return cleanup, nil
}

func copyFile(source, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open staged input: %w", err)
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create staged input: %w", err)
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return fmt.Errorf("copy staged input: %w", err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close staged input: %w", err)
	}
	return nil
}

func shouldIgnoreFile(path string) bool {
	base := filepath.Base(path)
	switch {
	case strings.HasSuffix(base, ".pyc"),
		strings.HasSuffix(base, ".pyo"),
		strings.HasSuffix(base, ".lock"),
		strings.HasSuffix(base, ".log"),
		strings.HasPrefix(base, "."):
		return true
	default:
		return false
	}
}

func isArtifactFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".html", ".pdf", ".csv", ".xlsx", ".xlsm", ".xls", ".ppt", ".pptx", ".docx", ".png", ".jpg", ".jpeg", ".webp", ".svg", ".zip":
		return true
	default:
		return false
	}
}

func detectArtifactKind(path string) ArtifactKind {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".html":
		return ArtifactKindMarkdown
	case ".csv", ".xlsx", ".xlsm", ".xls":
		return ArtifactKindSpreadsheet
	case ".pdf":
		return ArtifactKindPDF
	case ".ppt", ".pptx":
		return ArtifactKindPresentation
	case ".png", ".jpg", ".jpeg", ".webp", ".svg":
		return ArtifactKindImage
	case ".zip":
		return ArtifactKindArchive
	default:
		return ArtifactKindOther
	}
}

func detectArtifactPreviewState(path string) ArtifactPreviewState {
	switch detectArtifactKind(path) {
	case ArtifactKindMarkdown, ArtifactKindPDF, ArtifactKindImage:
		return ArtifactPreviewAvailable
	case ArtifactKindSpreadsheet, ArtifactKindPresentation:
		return ArtifactPreviewPending
	default:
		return ArtifactPreviewUnsupported
	}
}

func detectArtifactChangeType(exists bool) ArtifactChangeType {
	if exists {
		return ArtifactChangeModified
	}
	return ArtifactChangeCreated
}

func summarizeOutput(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= 240 {
		return trimmed
	}
	return string(runes[:240]) + "..."
}
