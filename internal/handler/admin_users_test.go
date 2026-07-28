package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type stubAdminListUserService struct {
	interfaces.UserService
	listAdminUsers func(ctx context.Context, q string, filter types.AdminUserListFilter, offset, limit int) ([]types.AdminUserListItem, int64, error)
}

func (s *stubAdminListUserService) ListAdminUsers(
	ctx context.Context, q string, filter types.AdminUserListFilter, offset, limit int,
) ([]types.AdminUserListItem, int64, error) {
	return s.listAdminUsers(ctx, q, filter, offset, limit)
}

func TestListAdminUsers_FilterClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotFilter types.AdminUserListFilter
	us := &stubAdminListUserService{
		listAdminUsers: func(_ context.Context, _ string, filter types.AdminUserListFilter, _, _ int) ([]types.AdminUserListItem, int64, error) {
			gotFilter = filter
			return []types.AdminUserListItem{{
				User: &types.User{
					ID:                  "u1",
					Username:            "client",
					Email:               "c@example.com",
					IsClientProvisioned: true,
				},
				HomeTenantName: "Client Workspace",
			}}, 1, nil
		},
	}
	h := &SystemHandler{userSvc: us}
	r := gin.New()
	r.GET("/system/admin/users", h.ListAdminUsers)

	req := httptest.NewRequest(http.MethodGet, "/system/admin/users?filter=client", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if gotFilter != types.AdminUserFilterClient {
		t.Fatalf("expected filter=client, got %q", gotFilter)
	}
	var body struct {
		Users []map[string]any `json:"users"`
		Total float64          `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Total != 1 || len(body.Users) != 1 {
		t.Fatalf("unexpected body: %+v", body)
	}
	if body.Users[0]["is_client_provisioned"] != true {
		t.Fatalf("expected client flag true: %+v", body.Users[0])
	}
	if body.Users[0]["home_tenant_name"] != "Client Workspace" {
		t.Fatalf("expected tenant name, got %+v", body.Users[0])
	}
}

func TestListAdminUsers_SearchQ(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotQ string
	us := &stubAdminListUserService{
		listAdminUsers: func(_ context.Context, q string, _ types.AdminUserListFilter, _, _ int) ([]types.AdminUserListItem, int64, error) {
			gotQ = q
			return nil, 0, nil
		},
	}
	h := &SystemHandler{userSvc: us}
	r := gin.New()
	r.GET("/system/admin/users", h.ListAdminUsers)

	req := httptest.NewRequest(http.MethodGet, "/system/admin/users?q=alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if gotQ != "alice" {
		t.Fatalf("expected q=alice, got %q", gotQ)
	}
}
