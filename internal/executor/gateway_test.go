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

	artifacts := detectArtifacts(basePath, "job-1", "session-1", pre, post, []string{filepath.Join(basePath, "input.md")})
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].RelativePath != "generated.md" {
		t.Fatalf("expected generated.md artifact, got %s", artifacts[0].RelativePath)
	}
}
