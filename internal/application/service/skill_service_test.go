package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeSkillForTest(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: Test skill " + name + "\n---\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "run.py"), []byte("print('ok')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSkillServiceDetailIncludesScriptsAndInstructions(t *testing.T) {
	root := t.TempDir()
	writeSkillForTest(t, root, "demo-skill", "Use `scripts/run.py` for execution.")
	t.Setenv("XELORA_SKILLS_DIR", root)

	svc := NewSkillService()
	detail, err := svc.GetSkillDetail(context.Background(), "demo-skill")
	if err != nil {
		t.Fatalf("GetSkillDetail failed: %v", err)
	}
	if detail.Name != "demo-skill" {
		t.Fatalf("name mismatch: %s", detail.Name)
	}
	if detail.Instructions == "" {
		t.Fatal("expected instructions")
	}
	if len(detail.Scripts) != 1 || detail.Scripts[0].Path != "scripts/run.py" {
		t.Fatalf("expected scripts/run.py, got %#v", detail.Scripts)
	}
}
