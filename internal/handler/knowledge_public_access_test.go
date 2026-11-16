package handler

import (
	"context"
	"errors"
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

// pubAccessDenyAllKBShare reports "not shared" for every lookup so these tests
// can prove that the public branch (or the guard result) is what granted
// access, rather than an organization share.
type pubAccessDenyAllKBShare struct {
	interfaces.KBShareService
}

func (pubAccessDenyAllKBShare) CheckTenantKBPermission(
	_ context.Context, _ string, _ uint64, _ types.TenantRole,
) (types.OrgMemberRole, bool, error) {
	return "", false, nil
}

func (pubAccessDenyAllKBShare) HasTenantKBPermission(
	_ context.Context, _ string, _ uint64, _ types.TenantRole, _ types.OrgMemberRole,
) (bool, error) {
	return false, nil
}

func (pubAccessDenyAllKBShare) GetKBSourceTenant(_ context.Context, _ string) (uint64, error) {
	return 0, errors.New("knowledge base is not shared")
}

// pubAccessGinContext builds a non-admin caller context for tenantID. Unlike
// systemAdminGinContext it never sets SystemAdminContextKey, so the handler
// must go through the ordinary visibility/share matrix.
func pubAccessGinContext(tenantID uint64, userID string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, tenantID)
	ctx = context.WithValue(ctx, types.UserIDContextKey, userID)
	c.Request = c.Request.WithContext(ctx)
	c.Set(types.TenantIDContextKey.String(), tenantID)
	c.Set(types.UserIDContextKey.String(), userID)
	return c, w
}

// A document inside a system-public knowledge base owned by another workspace
// must be readable. The KB list SQL, the RequireKBAccess guard, the KB detail
// handler and chat retrieval all honour visibility=public; denying it here
// breaks preview/download for everyone except the creator.
func TestResolveKnowledgeAndValidateKBAccess_PublicKBCrossTenant(t *testing.T) {
	const (
		callerTenant    = uint64(10011)
		resourceTenant  = uint64(10000)
		knowledgeID     = "knowledge-public"
		knowledgeBaseID = "kb-public"
	)
	h := &KnowledgeHandler{
		kgService: &systemAdminKGStub{knowledge: &types.Knowledge{
			ID:              knowledgeID,
			TenantID:        resourceTenant,
			KnowledgeBaseID: knowledgeBaseID,
		}},
		kbService: &systemAdminKBStub{kb: &types.KnowledgeBase{
			ID:         knowledgeBaseID,
			TenantID:   resourceTenant,
			Visibility: types.ResourceVisibilityPublic,
			CreatorID:  "another-user",
		}},
		kbShareService: pubAccessDenyAllKBShare{},
	}
	c, _ := pubAccessGinContext(callerTenant, "caller-user")

	knowledge, effectiveCtx, err := h.resolveKnowledgeAndValidateKBAccess(
		c, knowledgeID, types.OrgRoleViewer,
	)
	if err != nil {
		t.Fatalf("public KB document must be readable cross-workspace: %v", err)
	}
	if knowledge == nil || knowledge.ID != knowledgeID {
		t.Fatalf("returned knowledge = %#v, want ID %s", knowledge, knowledgeID)
	}
	if got, ok := types.TenantIDFromContext(effectiveCtx); !ok || got != resourceTenant {
		t.Fatalf("effective tenant = (%d, %v), want (%d, true)", got, ok, resourceTenant)
	}
	if got := c.GetUint64(types.TenantIDContextKey.String()); got != resourceTenant {
		t.Fatalf("Gin tenant = %d, want resource tenant %d", got, resourceTenant)
	}
}

// RequireKBAccess already resolved the full visibility/share matrix and
// rewrote the request tenant. The handler must reuse that decision instead of
// re-running a narrower check that rejects what the guard allowed - otherwise a
// workspace KB reached through an organization share or a shared agent still
// 403s at document level.
func TestResolveKnowledgeAndValidateKBAccess_HonoursKBAccessGuardResult(t *testing.T) {
	const (
		callerTenant    = uint64(10011)
		resourceTenant  = uint64(10000)
		knowledgeID     = "knowledge-shared"
		knowledgeBaseID = "kb-shared"
	)
	kb := &types.KnowledgeBase{
		ID:         knowledgeBaseID,
		TenantID:   resourceTenant,
		Visibility: types.ResourceVisibilityWorkspace,
		CreatorID:  "another-user",
	}
	h := &KnowledgeHandler{
		kgService: &systemAdminKGStub{knowledge: &types.Knowledge{
			ID:              knowledgeID,
			TenantID:        resourceTenant,
			KnowledgeBaseID: knowledgeBaseID,
		}},
		kbService:      &systemAdminKBStub{kb: kb},
		kbShareService: pubAccessDenyAllKBShare{},
	}
	c, _ := pubAccessGinContext(callerTenant, "caller-user")
	c.Set(middleware.KBAccessContextKey, &middleware.KBAccess{
		KnowledgeBase:     kb,
		EffectiveTenantID: resourceTenant,
		Permission:        types.OrgRoleViewer,
	})

	knowledge, effectiveCtx, err := h.resolveKnowledgeAndValidateKBAccess(
		c, knowledgeID, types.OrgRoleViewer,
	)
	if err != nil {
		t.Fatalf("guard-resolved access must not be re-denied by the handler: %v", err)
	}
	if knowledge == nil || knowledge.ID != knowledgeID {
		t.Fatalf("returned knowledge = %#v, want ID %s", knowledge, knowledgeID)
	}
	if got, ok := types.TenantIDFromContext(effectiveCtx); !ok || got != resourceTenant {
		t.Fatalf("effective tenant = (%d, %v), want (%d, true)", got, ok, resourceTenant)
	}
	if got := c.GetUint64(types.TenantIDContextKey.String()); got != resourceTenant {
		t.Fatalf("Gin tenant = %d, want resource tenant %d", got, resourceTenant)
	}
}

// Public access is read-only. An Editor-level requirement must still be
// rejected so the fix cannot become a write bypass on another workspace's
// documents.
func TestResolveKnowledgeAndValidateKBAccess_PublicKBDeniesEditor(t *testing.T) {
	const (
		callerTenant    = uint64(10011)
		resourceTenant  = uint64(10000)
		knowledgeID     = "knowledge-public"
		knowledgeBaseID = "kb-public"
	)
	h := &KnowledgeHandler{
		kgService: &systemAdminKGStub{knowledge: &types.Knowledge{
			ID:              knowledgeID,
			TenantID:        resourceTenant,
			KnowledgeBaseID: knowledgeBaseID,
		}},
		kbService: &systemAdminKBStub{kb: &types.KnowledgeBase{
			ID:         knowledgeBaseID,
			TenantID:   resourceTenant,
			Visibility: types.ResourceVisibilityPublic,
			CreatorID:  "another-user",
		}},
		kbShareService: pubAccessDenyAllKBShare{},
	}
	c, _ := pubAccessGinContext(callerTenant, "caller-user")

	if _, _, err := h.resolveKnowledgeAndValidateKBAccess(
		c, knowledgeID, types.OrgRoleEditor,
	); err == nil {
		t.Fatal("editor access to a foreign public KB document must be denied")
	}
}
