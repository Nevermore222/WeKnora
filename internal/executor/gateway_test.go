package executor

import (
	"path/filepath"
	"testing"
	"time"
)

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
