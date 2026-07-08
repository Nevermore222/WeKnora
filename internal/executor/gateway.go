package executor

import (
	"context"
	"mime"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Tencent/Xelora/internal/agent/skills"
	"github.com/Tencent/Xelora/internal/sandbox"
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
	providers map[string]Provider
}

func NewGateway() *Gateway {
	return NewGatewayWithProviders(
		NewLocalProvider(),
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
	return &Gateway{providers: registry}
}

func (g *Gateway) RunSkillScriptJob(ctx context.Context, req SkillJobRequest, executor SkillExecutor) (*SkillJobExecution, error) {
	basePath, err := executor.GetSkillBasePath(ctx, req.SkillName)
	if err != nil {
		return nil, err
	}

	provider, err := selectProvider(req.Provider, g.providers)
	if err != nil {
		return nil, err
	}
	providerCapability := provider.Capability(ctx)

	now := time.Now()
	workspace := buildWorkspaceRecord(req, basePath, provider.Name(), now)

	preSnapshot, err := snapshotFiles(basePath)
	if err != nil {
		return nil, err
	}

	job := &SkillJob{
		ID:                 uuid.NewString(),
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

	outcome, err := provider.ExecuteSkillScript(ctx, req, prepared, executor)
	if err != nil {
		return nil, err
	}

	postSnapshot, snapshotErr := snapshotFiles(basePath)
	if snapshotErr == nil {
		artifacts := detectArtifacts(basePath, workspace.ID, job.ID, req.SessionID, preSnapshot, postSnapshot, outcome.MaterializedInputPaths)
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
	case "node_modules", "__pycache__", ".git":
		return true
	default:
		return false
	}
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
