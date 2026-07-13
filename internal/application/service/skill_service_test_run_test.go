package service

import (
	"context"
	"strings"
	"testing"

	"github.com/Tencent/Xelora/internal/agent/skills"
)

func TestSkillServiceTestRunRequiresWorkspace(t *testing.T) {
	root := t.TempDir()
	writeSkillForTest(t, root, "demo-skill", "Use `scripts/run.py` for execution.")
	t.Setenv("XELORA_SKILLS_DIR", root)

	svc := NewSkillService()
	result, err := svc.TestRunSkill(context.Background(), "demo-skill", skills.SkillTestRunRequest{
		ScriptPath: "scripts/run.py",
		Args:       []string{"request.json"},
	})
	if err != nil {
		t.Fatalf("TestRunSkill failed: %v", err)
	}
	if result.SkillName != "demo-skill" || result.ScriptPath != "scripts/run.py" {
		t.Fatalf("unexpected result identity: %#v", result)
	}
	if result.Success {
		t.Fatalf("test run should not execute without workspace: %#v", result)
	}
	if !strings.Contains(result.Error, "workspace_required") {
		t.Fatalf("expected workspace_required error, got %q", result.Error)
	}
}
