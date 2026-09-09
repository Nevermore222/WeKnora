package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/Xelora/internal/middleware"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
)

type stubKnowledgeHandlerKBService struct {
	interfaces.KnowledgeBaseService
	called bool
}

func (s *stubKnowledgeHandlerKBService) GetKnowledgeBaseByID(context.Context, string) (*types.KnowledgeBase, error) {
	s.called = true
	return nil, nil
}

func TestValidateKnowledgeBaseAccess_UsesResolvedPublicAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/knowledge-bases/kb-public/knowledge", nil)
	c.Params = gin.Params{{Key: "id", Value: "kb-public"}}
	c.Set(types.TenantIDContextKey.String(), uint64(100))
	c.Set(types.UserIDContextKey.String(), "user-100")

	kb := &types.KnowledgeBase{
		ID:         "kb-public",
		TenantID:   200,
		CreatorID:  "owner-200",
		Visibility: types.ResourceVisibilityPublic,
	}
	c.Set(middleware.KBAccessContextKey, &middleware.KBAccess{
		KnowledgeBase:     kb,
		EffectiveTenantID: 200,
		Permission:        types.OrgRoleViewer,
	})

	lookup := &stubKnowledgeHandlerKBService{}
	h := &KnowledgeHandler{kbService: lookup}
	got, id, effectiveTenantID, permission, err := h.validateKnowledgeBaseAccess(c)
	if err != nil {
		t.Fatalf("expected resolved public KB to pass, got error: %v", err)
	}
	if got != kb || id != kb.ID || effectiveTenantID != 200 || permission != types.OrgRoleViewer {
		t.Fatalf("unexpected access result: kb=%p id=%q tenant=%d permission=%q", got, id, effectiveTenantID, permission)
	}
	if lookup.called {
		t.Fatal("handler should reuse the middleware access result instead of looking up the KB again")
	}
}
