package executor

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Tencent/Xelora/internal/agent/skills"
	"github.com/Tencent/Xelora/internal/sandbox"
)

type fakeSkillExecutor struct {
	basePath string
	outcome  *skills.ScriptExecutionOutcome
	err      error
}

func (f *fakeSkillExecutor) GetSkillBasePath(ctx context.Context, skillName string) (string, error) {
	return f.basePath, nil
}

func (f *fakeSkillExecutor) ExecuteScriptDetailed(ctx context.Context, skillName, scriptPath string, args []string, stdin string) (*skills.ScriptExecutionOutcome, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.outcome, nil
}

func TestDetectArtifactsIgnoresMaterializedInput(t *testing.T) {
	basePath := filepath.Clean("/tmp/skill")
	now := time.Now()
	pre := map[string]fileSnapshot{
		"existing.md": {
			RelativePath: "existing.md",
			AbsolutePath: filepath.Join(basePath, "existing.md"),
			Size:         5,
			ModifiedAt:   now,
		},
	}
	post := map[string]fileSnapshot{
		"generated.md": {
			RelativePath: "generated.md",
			AbsolutePath: filepath.Join(basePath, "generated.md"),
			Size:         9,
			ModifiedAt:   now.Add(time.Second),
		},
		"input.md": {
			RelativePath: "input.md",
			AbsolutePath: filepath.Join(basePath, "input.md"),
			Size:         10,
			ModifiedAt:   now.Add(time.Second),
		},
	}

	artifacts := detectArtifacts(basePath, "workspace-1", "job-1", "session-1", pre, post, []string{filepath.Join(basePath, "input.md")})
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	artifact := artifacts[0]
	if artifact.RelativePath != "generated.md" {
		t.Fatalf("expected generated.md artifact, got %s", artifact.RelativePath)
	}
	if artifact.WorkspaceID != "workspace-1" {
		t.Fatalf("expected workspace-1, got %s", artifact.WorkspaceID)
	}
	if artifact.Kind != ArtifactKindMarkdown {
		t.Fatalf("expected markdown kind, got %s", artifact.Kind)
	}
	if artifact.PreviewState != ArtifactPreviewAvailable {
		t.Fatalf("expected available preview, got %s", artifact.PreviewState)
	}
	if artifact.ChangeType != ArtifactChangeCreated {
		t.Fatalf("expected created change type, got %s", artifact.ChangeType)
	}
}

func TestBuildWorkspaceIDUsesSessionAndSkill(t *testing.T) {
	id := buildWorkspaceID("Session:123", "baoyu-format-markdown")
	want := "session-session-123-skill-baoyu-format-markdown"
	if id != want {
		t.Fatalf("expected %s, got %s", want, id)
	}
}

func TestRunSkillScriptJobDefaultsToLocalProvider(t *testing.T) {
	basePath := t.TempDir()
	gateway := NewGateway()
	executor := &fakeSkillExecutor{
		basePath: basePath,
		outcome: &skills.ScriptExecutionOutcome{
			Result: &sandbox.ExecuteResult{
				ExitCode: 0,
				Duration: 25 * time.Millisecond,
				Stdout:   "ok",
			},
			BasePath:               basePath,
			MaterializedInputPaths: nil,
		},
	}

	execution, err := gateway.RunSkillScriptJob(context.Background(), SkillJobRequest{
		SessionID:  "session-1",
		SkillName:  "demo-skill",
		ScriptPath: "scripts/demo.py",
	}, executor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execution.Job.Provider != LocalProviderName {
		t.Fatalf("expected provider %s, got %s", LocalProviderName, execution.Job.Provider)
	}
	if execution.Workspace.Provider != LocalProviderName {
		t.Fatalf("expected workspace provider %s, got %s", LocalProviderName, execution.Workspace.Provider)
	}
	if execution.Provider.Provider != LocalProviderName {
		t.Fatalf("expected provider capability for %s, got %s", LocalProviderName, execution.Provider.Provider)
	}
	if execution.Provider.Status != ProviderStatusAvailable {
		t.Fatalf("expected provider status available, got %s", execution.Provider.Status)
	}
	if execution.Job.Status != JobStatusSucceeded {
		t.Fatalf("expected succeeded job, got %s", execution.Job.Status)
	}
	if execution.Job.WorkspaceID == "" {
		t.Fatal("expected workspace id to be populated")
	}
	if execution.Job.WorkspaceID != execution.Workspace.ID {
		t.Fatalf("expected workspace ids to match, got %s and %s", execution.Job.WorkspaceID, execution.Workspace.ID)
	}
	if execution.ArtifactDetected {
		t.Fatal("expected no artifacts for empty workspace execution")
	}
}

func TestRunSkillScriptJobRejectsUnknownProvider(t *testing.T) {
	basePath := t.TempDir()
	gateway := NewGateway()
	executor := &fakeSkillExecutor{basePath: basePath}

	_, err := gateway.RunSkillScriptJob(context.Background(), SkillJobRequest{
		Provider:   "cubesandbox",
		SkillName:  "demo-skill",
		ScriptPath: "scripts/demo.py",
	}, executor)
	if err == nil {
		t.Fatal("expected unknown provider error")
	}
	if got := err.Error(); got != "executor provider not configured: cubesandbox" {
		t.Fatalf("unexpected error: %s", got)
	}
}
