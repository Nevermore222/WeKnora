package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
)

type callerTenantUserServiceStub struct {
	interfaces.UserService
	user *types.User
}

func (s *callerTenantUserServiceStub) ValidateToken(
	_ context.Context, _ string,
) (*types.User, uint64, error) {
	return s.user, s.user.TenantID, nil
}

type callerTenantServiceStub struct {
	interfaces.TenantService
	tenants map[uint64]*types.Tenant
}

func (s *callerTenantServiceStub) GetTenantByID(_ context.Context, id uint64) (*types.Tenant, error) {
	return s.tenants[id], nil
}

func TestAuth_PreservesCallerTenantBesideOperationTenant(t *testing.T) {
	const tenantID = uint64(10011)
	userSvc := &callerTenantUserServiceStub{user: &types.User{
		ID:            "admin-user",
		TenantID:      tenantID,
		IsSystemAdmin: true,
	}}
	tenantSvc := &callerTenantServiceStub{tenants: map[uint64]*types.Tenant{
		tenantID: {ID: tenantID},
	}}
	memberSvc := newFakeMemberService()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Auth(tenantSvc, userSvc, memberSvc, cfgWithRBAC(true)))
	r.GET("/", func(c *gin.Context) {
		callerTenantID, callerOK := c.Request.Context().Value(types.CallerTenantIDContextKey).(uint64)
		operationTenantID, operationOK := types.TenantIDFromContext(c.Request.Context())
		if !callerOK || callerTenantID != tenantID {
			t.Errorf("caller tenant key = (%d, %v), want (%d, true)", callerTenantID, callerOK, tenantID)
		}
		if !operationOK || operationTenantID != tenantID {
			t.Errorf("operation tenant = (%d, %v), want (%d, true)", operationTenantID, operationOK, tenantID)
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusNoContent, w.Body.String())
	}
}
