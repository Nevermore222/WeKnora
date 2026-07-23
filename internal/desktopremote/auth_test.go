package desktopremote

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fixedProfileSource struct {
	profile *ServerProfile
	err     error
}

func (s fixedProfileSource) Get(context.Context, string) (*ServerProfile, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.profile, nil
}

func TestLoginStoresRefreshTokenAndReturnsRedactedIdentity(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"user": {"id":"u1","email":"u1@example.com"},
			"active_tenant": {"id":42,"name":"Workspace"},
			"memberships": [{"tenant_id":42,"role":"owner"}],
			"token": "access-login",
			"refresh_token": "refresh-login"
		}`))
	}))
	defer upstream.Close()

	credentials := newFakeCredentialStore()
	sessions := NewSessionStore()
	auth := NewAuthClient(fixedProfileSource{profile: &ServerProfile{
		ID:                     "server-a",
		BaseURL:                upstream.URL,
		AllowInsecureTransport: true,
	}}, credentials, sessions, upstream.Client())

	snapshot, err := auth.Login(context.Background(), "server-a", strings.NewReader(`{"email":"u1@example.com","password":"secret"}`))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "access-login") || strings.Contains(string(raw), "refresh-login") {
		t.Fatalf("identity leaked token: %s", raw)
	}
	if !snapshot.Authenticated || snapshot.UserID != "u1" || snapshot.TenantID != 42 {
		t.Fatalf("snapshot=%+v", snapshot)
	}

	token, err := credentials.GetRefreshToken(context.Background(), "server-a", "u1")
	if err != nil || token != "refresh-login" {
		t.Fatalf("refresh token = %q, %v", token, err)
	}
	session, ok := sessions.Get("server-a")
	if !ok || session.AccessToken != "access-login" || session.RefreshTokenOwner != "u1" {
		t.Fatalf("session=%+v ok=%v", session, ok)
	}
}

func TestAccessTokenRefreshesExpiredSessionAndRotatesCredential(t *testing.T) {
	refreshCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/refresh" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		refreshCalls++
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["refreshToken"] != "refresh-old" {
			t.Fatalf("refresh body=%v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"access_token":"access-new","refresh_token":"refresh-new"}`))
	}))
	defer upstream.Close()

	credentials := newFakeCredentialStore()
	if err := credentials.PutRefreshToken(context.Background(), "server-a", "u1", "refresh-old"); err != nil {
		t.Fatal(err)
	}
	sessions := NewSessionStore()
	sessions.Set("server-a", &Session{
		UserID:            "u1",
		AccessToken:       "access-old",
		RefreshTokenOwner: "u1",
		AccessExpiresAt:   time.Now().Add(-time.Minute),
		Snapshot:          IdentitySnapshot{Authenticated: true, UserID: "u1"},
	})
	auth := NewAuthClient(fixedProfileSource{profile: &ServerProfile{
		ID:                     "server-a",
		BaseURL:                upstream.URL,
		AllowInsecureTransport: true,
	}}, credentials, sessions, upstream.Client())

	token, err := auth.AccessToken(context.Background(), "server-a")
	if err != nil {
		t.Fatal(err)
	}
	if token != "access-new" || refreshCalls != 1 {
		t.Fatalf("token=%q refreshCalls=%d", token, refreshCalls)
	}
	stored, err := credentials.GetRefreshToken(context.Background(), "server-a", "u1")
	if err != nil || stored != "refresh-new" {
		t.Fatalf("stored refresh = %q, %v", stored, err)
	}

	token, err = auth.AccessToken(context.Background(), "server-a")
	if err != nil {
		t.Fatal(err)
	}
	if token != "access-new" || refreshCalls != 1 {
		t.Fatalf("unexpected second refresh token=%q calls=%d", token, refreshCalls)
	}
}

func TestLogoutRemovesSessionAndCredential(t *testing.T) {
	credentials := newFakeCredentialStore()
	if err := credentials.PutRefreshToken(context.Background(), "server-a", "u1", "refresh"); err != nil {
		t.Fatal(err)
	}
	sessions := NewSessionStore()
	sessions.Set("server-a", &Session{
		UserID:            "u1",
		AccessToken:       "access",
		RefreshTokenOwner: "u1",
		Snapshot:          IdentitySnapshot{Authenticated: true, UserID: "u1"},
	})
	auth := NewAuthClient(fixedProfileSource{profile: &ServerProfile{ID: "server-a"}}, credentials, sessions, nil)

	if err := auth.Logout(context.Background(), "server-a"); err != nil {
		t.Fatal(err)
	}
	if sessions.Snapshot("server-a").Authenticated {
		t.Fatal("session still authenticated")
	}
	_, err := credentials.GetRefreshToken(context.Background(), "server-a", "u1")
	if !errors.Is(err, ErrCredentialNotFound) {
		t.Fatalf("credential error=%v", err)
	}
}
