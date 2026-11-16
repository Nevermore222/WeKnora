package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/gin-gonic/gin"
)

func capabilityHarness(user *types.User, mw gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			ctx := context.WithValue(c.Request.Context(), types.UserContextKey, user)
			ctx = context.WithValue(ctx, types.UserIDContextKey, user.ID)
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	})
	r.GET("/protected", mw, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)
	return w
}

func TestRequireCapability_DeniesWhenFalse(t *testing.T) {
	w := capabilityHarness(&types.User{
		ID:                     "u1",
		CanCreateKnowledgeBase: false,
	}, RequireCapability(types.CapabilityCreateKnowledgeBase))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["code"] != "capability_denied" {
		t.Fatalf("expected capability_denied, got %#v", body)
	}
}

func TestRequireCapability_AllowsWhenTrue(t *testing.T) {
	w := capabilityHarness(&types.User{
		ID:                     "u1",
		CanCreateKnowledgeBase: true,
	}, RequireCapability(types.CapabilityCreateKnowledgeBase))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRequireCapability_AllowsSystemAdminForRegisteredCapability(t *testing.T) {
	w := capabilityHarness(&types.User{
		ID:                     "root",
		IsSystemAdmin:          true,
		CanCreateKnowledgeBase: false,
	}, RequireCapability(types.CapabilityCreateKnowledgeBase))
	if w.Code != http.StatusOK {
		t.Fatalf("system admin must bypass capability gate, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestUserHasCapability_SystemAdminOnlyBypassesRegisteredIDs(t *testing.T) {
	user := &types.User{ID: "root", IsSystemAdmin: true}
	if !types.UserHasCapability(user, types.CapabilityCreateKnowledgeBase) {
		t.Fatalf("system admin must have every registered capability")
	}
	if types.UserHasCapability(user, "not_registered") {
		t.Fatalf("system admin bypass must not make unknown capability IDs valid")
	}
}

func TestRequireCapability_DeniesNilUser(t *testing.T) {
	w := capabilityHarness(nil, RequireCapability(types.CapabilityCreateAgent))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for nil user, got %d", w.Code)
	}
}
