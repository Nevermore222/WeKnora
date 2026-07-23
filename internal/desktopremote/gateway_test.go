package desktopremote

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGatewayLocksTargetAndRebuildsHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/agents" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.URL.RawQuery != "page=1" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Tenant-ID") != "42" {
			t.Fatalf("tenant=%q", r.Header.Get("X-Tenant-ID"))
		}
		if r.Header.Get("X-Forwarded-Host") != "" {
			t.Fatal("forwarding header leaked")
		}
		if r.Header.Get("X-Not-Allowed") != "" {
			t.Fatal("non-allowlisted header leaked")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	gateway := newGatewayForTest(upstream.URL, &Session{
		UserID:            "u1",
		AccessToken:       "access-token",
		RefreshTokenOwner: "u1",
		AccessExpiresAt:   time.Now().Add(time.Hour),
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/enterprise/server-a/proxy/api/v1/agents?page=1", nil)
	req.Header.Set("Authorization", "Bearer caller-token")
	req.Header.Set("X-Tenant-ID", "42")
	req.Header.Set("X-Forwarded-Host", "evil.example")
	req.Header.Set("X-Not-Allowed", "leak")
	rec := httptest.NewRecorder()

	gateway.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "application/json" || rec.Header().Get("ETag") != `"v1"` {
		t.Fatalf("headers=%v", rec.Header())
	}
	if rec.Body.String() != `{"ok":true}` {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestGatewayRejectsMissingSessionAndExternalTargets(t *testing.T) {
	gateway := newGatewayForTest("http://127.0.0.1:1", nil)

	rec := httptest.NewRecorder()
	gateway.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/desktop/enterprise/server-a/proxy/api/v1/agents", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing session status=%d", rec.Code)
	}

	rec = httptest.NewRecorder()
	gateway.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/desktop/enterprise/server-a/proxy//evil.example/api", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("scheme-relative status=%d", rec.Code)
	}
}

func TestGatewayPreservesRequestBodyAndRangeResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != `{"name":"doc"}` {
			t.Fatalf("body=%s", body)
		}
		if r.Header.Get("Range") != "bytes=0-3" {
			t.Fatalf("range=%q", r.Header.Get("Range"))
		}
		w.Header().Set("Content-Range", "bytes 0-3/8")
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("data"))
	}))
	defer upstream.Close()

	gateway := newGatewayForTest(upstream.URL, &Session{
		UserID:            "u1",
		AccessToken:       "access-token",
		RefreshTokenOwner: "u1",
		AccessExpiresAt:   time.Now().Add(time.Hour),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/enterprise/server-a/proxy/api/v1/knowledge", strings.NewReader(`{"name":"doc"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Range", "bytes=0-3")
	rec := httptest.NewRecorder()

	gateway.ServeHTTP(rec, req)

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Range") != "bytes 0-3/8" || rec.Body.String() != "data" {
		t.Fatalf("headers=%v body=%s", rec.Header(), rec.Body.String())
	}
}

func TestGatewayRefreshesAndReplaysGetOnceAfterUnauthorized(t *testing.T) {
	agentCalls := 0
	refreshCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/agents":
			agentCalls++
			if agentCalls == 1 {
				if r.Header.Get("Authorization") != "Bearer access-old" {
					t.Fatalf("first auth=%q", r.Header.Get("Authorization"))
				}
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if r.Header.Get("Authorization") != "Bearer access-new" {
				t.Fatalf("second auth=%q", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/api/v1/auth/refresh":
			refreshCalls++
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["refreshToken"] != "refresh-old" {
				t.Fatalf("refresh body=%v", body)
			}
			_, _ = w.Write([]byte(`{"success":true,"access_token":"access-new","refresh_token":"refresh-new"}`))
		default:
			t.Fatalf("path=%s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	gateway, credentials := newGatewayWithAuthForTest(upstream.URL, &Session{
		UserID:            "u1",
		AccessToken:       "access-old",
		RefreshTokenOwner: "u1",
		AccessExpiresAt:   time.Now().Add(time.Hour),
		Snapshot:          IdentitySnapshot{Authenticated: true, UserID: "u1"},
	})
	if err := credentials.PutRefreshToken(t.Context(), "server-a", "u1", "refresh-old"); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	gateway.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/desktop/enterprise/server-a/proxy/api/v1/agents", nil))

	if rec.Code != http.StatusOK || agentCalls != 2 || refreshCalls != 1 {
		t.Fatalf("status=%d agents=%d refresh=%d body=%s", rec.Code, agentCalls, refreshCalls, rec.Body.String())
	}
	stored, err := credentials.GetRefreshToken(t.Context(), "server-a", "u1")
	if err != nil || stored != "refresh-new" {
		t.Fatalf("stored refresh=%q err=%v", stored, err)
	}
}

func TestGatewayDoesNotReplayUnsafePostWithoutIdempotencyKey(t *testing.T) {
	agentCalls := 0
	refreshCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/agents":
			agentCalls++
			w.WriteHeader(http.StatusUnauthorized)
		case "/api/v1/auth/refresh":
			refreshCalls++
			_, _ = w.Write([]byte(`{"success":true,"access_token":"access-new","refresh_token":"refresh-new"}`))
		default:
			t.Fatalf("path=%s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	gateway, credentials := newGatewayWithAuthForTest(upstream.URL, &Session{
		UserID:            "u1",
		AccessToken:       "access-old",
		RefreshTokenOwner: "u1",
		AccessExpiresAt:   time.Now().Add(time.Hour),
		Snapshot:          IdentitySnapshot{Authenticated: true, UserID: "u1"},
	})
	if err := credentials.PutRefreshToken(t.Context(), "server-a", "u1", "refresh-old"); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/enterprise/server-a/proxy/api/v1/agents", strings.NewReader(`{"name":"agent"}`))
	gateway.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized || agentCalls != 1 || refreshCalls != 0 {
		t.Fatalf("status=%d agents=%d refresh=%d", rec.Code, agentCalls, refreshCalls)
	}
}

func TestGatewayReplaysPostWithIdempotencyKey(t *testing.T) {
	agentCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/agents":
			agentCalls++
			if agentCalls == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if r.Header.Get("Idempotency-Key") != "create-1" {
				t.Fatalf("idempotency=%q", r.Header.Get("Idempotency-Key"))
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != `{"name":"agent"}` {
				t.Fatalf("body=%s", body)
			}
			w.WriteHeader(http.StatusCreated)
		case "/api/v1/auth/refresh":
			_, _ = w.Write([]byte(`{"success":true,"access_token":"access-new","refresh_token":"refresh-new"}`))
		default:
			t.Fatalf("path=%s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	gateway, credentials := newGatewayWithAuthForTest(upstream.URL, &Session{
		UserID:            "u1",
		AccessToken:       "access-old",
		RefreshTokenOwner: "u1",
		AccessExpiresAt:   time.Now().Add(time.Hour),
		Snapshot:          IdentitySnapshot{Authenticated: true, UserID: "u1"},
	})
	if err := credentials.PutRefreshToken(t.Context(), "server-a", "u1", "refresh-old"); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/enterprise/server-a/proxy/api/v1/agents", strings.NewReader(`{"name":"agent"}`))
	req.Header.Set("Idempotency-Key", "create-1")
	gateway.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated || agentCalls != 2 {
		t.Fatalf("status=%d agents=%d body=%s", rec.Code, agentCalls, rec.Body.String())
	}
}

func newGatewayForTest(baseURL string, session *Session) *Gateway {
	sessions := NewSessionStore()
	if session != nil {
		sessions.Set("server-a", session)
	}
	return NewGateway(fixedProfileSource{profile: &ServerProfile{
		ID:                     "server-a",
		BaseURL:                baseURL,
		AllowInsecureTransport: true,
	}}, sessions, http.DefaultClient)
}

func newGatewayWithAuthForTest(baseURL string, session *Session) (*Gateway, *fakeCredentialStore) {
	sessions := NewSessionStore()
	if session != nil {
		sessions.Set("server-a", session)
	}
	profiles := fixedProfileSource{profile: &ServerProfile{
		ID:                     "server-a",
		BaseURL:                baseURL,
		AllowInsecureTransport: true,
	}}
	credentials := newFakeCredentialStore()
	auth := NewAuthClient(profiles, credentials, sessions, http.DefaultClient)
	return NewGatewayWithAuth(profiles, sessions, auth, http.DefaultClient), credentials
}
