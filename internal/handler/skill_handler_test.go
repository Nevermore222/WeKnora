package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/Xelora/internal/agent/skills"
	"github.com/Tencent/Xelora/internal/middleware"
	"github.com/gin-gonic/gin"
)

type stubSkillService struct {
	testRun func(context.Context, string, skills.SkillTestRunRequest) (*skills.SkillTestRunResult, error)
}

func (s *stubSkillService) ListPreloadedSkills(context.Context) ([]*skills.SkillMetadata, error) {
	return nil, nil
}

func (s *stubSkillService) ListSkillSummaries(context.Context) ([]*skills.SkillSummary, error) {
	return nil, nil
}

func (s *stubSkillService) GetSkillByName(context.Context, string) (*skills.Skill, error) {
	return nil, nil
}

func (s *stubSkillService) GetSkillDetail(context.Context, string) (*skills.SkillDetail, error) {
	return nil, nil
}

func (s *stubSkillService) GetSkillFile(context.Context, string, string) (*skills.SkillFile, error) {
	return nil, nil
}

func (s *stubSkillService) TestRunSkill(ctx context.Context, name string, req skills.SkillTestRunRequest) (*skills.SkillTestRunResult, error) {
	if s.testRun == nil {
		return nil, nil
	}
	return s.testRun(ctx, name, req)
}

func TestSkillHandlerTestRunRequiresScriptPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler())
	handler := NewSkillHandler(&stubSkillService{
		testRun: func(context.Context, string, skills.SkillTestRunRequest) (*skills.SkillTestRunResult, error) {
			t.Fatal("service should not be called when script_path is missing")
			return nil, nil
		},
	})
	router.POST("/skills/:name/test-run", handler.TestRunSkill)

	body := bytes.NewBufferString(`{"args":["request.json"]}`)
	req := httptest.NewRequest(http.MethodPost, "/skills/officecli-document-editing/test-run", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "script_path") {
		t.Fatalf("error should mention script_path, got %s", w.Body.String())
	}
}

func TestSkillHandlerTestRunReturnsStructuredResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler())
	handler := NewSkillHandler(&stubSkillService{
		testRun: func(_ context.Context, name string, req skills.SkillTestRunRequest) (*skills.SkillTestRunResult, error) {
			if name != "officecli-document-editing" {
				t.Fatalf("skill name: got %s", name)
			}
			if req.ScriptPath != "scripts/officecli_bridge.py" {
				t.Fatalf("script path: got %s", req.ScriptPath)
			}
			return &skills.SkillTestRunResult{
				SkillName:  name,
				ScriptPath: req.ScriptPath,
				Args:       req.Args,
				Success:    false,
				Error:      "workspace_required: bind a workspace before running skill test scripts",
				Artifacts:  []skills.SkillTestRunArtifact{},
			}, nil
		},
	})
	router.POST("/skills/:name/test-run", handler.TestRunSkill)

	payload := map[string]any{
		"script_path": "scripts/officecli_bridge.py",
		"args":        []string{"request.json"},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/skills/officecli-document-editing/test-run", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "workspace_required") {
		t.Fatalf("response should include workspace_required, got %s", w.Body.String())
	}
}
