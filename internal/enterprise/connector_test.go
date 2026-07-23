package enterprise

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProvisionClientUserParsesHomeTenantID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tenants/10001/client-users" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "api-token" {
			t.Fatalf("unexpected API token: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": {
				"user_id": "u-client",
				"email": "client@example.com",
				"tenant_id": 20002
			}
		}`))
	}))
	defer server.Close()

	provisioned, err := provisionClientUser(
		context.Background(),
		server.Client(),
		server.URL,
		"api-token",
		"10001",
		"client@example.com",
		"client",
		"secret",
	)
	if err != nil {
		t.Fatalf("provisionClientUser returned error: %v", err)
	}
	if provisioned.UserID != "u-client" || provisioned.Email != "client@example.com" {
		t.Fatalf("unexpected provisioned user: %+v", provisioned)
	}
	if provisioned.TenantID != "20002" {
		t.Fatalf("expected home tenant id 20002, got %q", provisioned.TenantID)
	}
}
