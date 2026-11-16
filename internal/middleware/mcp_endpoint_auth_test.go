package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type mcpEndpointAuthServiceStub struct {
	interfaces.MCPEndpointService
	ep *types.MCPEndpoint
}

func (s *mcpEndpointAuthServiceStub) Authenticate(context.Context, string, string) (*types.MCPEndpoint, error) {
	return s.ep, nil
}

type mcpEndpointTenantServiceStub struct {
	interfaces.TenantService
	tenant *types.Tenant
}

func (s *mcpEndpointTenantServiceStub) GetTenantByID(context.Context, uint64) (*types.Tenant, error) {
	return s.tenant, nil
}

func TestMCPEndpointAuthInstallsXeloraContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ep := &types.MCPEndpoint{ID: "ep-1", TenantID: 7, Enabled: true}
	r := gin.New()
	r.POST("/mcp/:endpoint_id", MCPEndpointAuth(
		&mcpEndpointAuthServiceStub{ep: ep},
		&mcpEndpointTenantServiceStub{tenant: &types.Tenant{ID: 7, Status: "active"}},
	), func(c *gin.Context) {
		tenantID, ok := types.TenantIDFromContext(c.Request.Context())
		if !ok || tenantID != 7 {
			t.Errorf("tenant context = %d/%v", tenantID, ok)
		}
		if got, ok := MCPEndpointFromContext(c.Request.Context()); !ok || got.ID != "ep-1" {
			t.Errorf("endpoint context = %#v/%v", got, ok)
		}
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp/ep-1", nil)
	req.Header.Set("Authorization", "Bearer mcp_test")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}
