package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/Xelora/internal/desktopremote"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDesktopRemoteRoutesRequireSessionAndProxyServerAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/system/capabilities":
			_, _ = w.Write([]byte(`{"api_contract_major":1,"api_contract_minor":0,"server_version":"test","features":["organizations"]}`))
		case "/api/v1/auth/login":
			_, _ = w.Write([]byte(`{
				"success": true,
				"user": {"id":"u1","email":"u1@example.com"},
				"active_tenant": {"id":42,"name":"Workspace"},
				"memberships": [{"tenant_id":42,"role":"owner"}],
				"token": "access-token",
				"refresh_token": "refresh-token"
			}`))
		case "/api/v1/auth/config":
			if r.Header.Get("Authorization") != "" {
				t.Fatalf("auth config leaked authorization=%q", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"registration_mode":"open"}`))
		case "/api/v1/auth/register":
			if r.Header.Get("Authorization") != "" {
				t.Fatalf("register leaked authorization=%q", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"user":{"id":"u2","email":"u2@example.com"},"tenant":{"id":"43","name":"New"}}}`))
		case "/api/v1/agents":
			if r.Header.Get("Authorization") != "Bearer access-token" {
				t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"agents":[]}`))
		default:
			t.Fatalf("path=%s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	handler := newDesktopRemoteHandlerForTest(t)
	router := gin.New()
	group := router.Group("/desktop/remote", handler.RequireDesktopSession())
	handler.RegisterRoutes(group)

	blocked := httptest.NewRecorder()
	router.ServeHTTP(blocked, httptest.NewRequest(http.MethodPost, "/desktop/remote/profiles", nil))
	if blocked.Code != http.StatusUnauthorized {
		t.Fatalf("missing session status=%d", blocked.Code)
	}

	createBody := []byte(`{"name":"admin","base_url":` + strconvQuote(upstream.URL) + `,"allow_insecure_transport":true}`)
	create := httptest.NewRecorder()
	createReq := desktopRequest(http.MethodPost, "/desktop/remote/profiles", createBody)
	router.ServeHTTP(create, createReq)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var profile desktopremote.ServerProfile
	if err := json.Unmarshal(create.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}

	activate := httptest.NewRecorder()
	router.ServeHTTP(activate, desktopRequest(http.MethodPost, "/desktop/remote/profiles/"+profile.ID+"/activate", nil))
	if activate.Code != http.StatusOK {
		t.Fatalf("activate status=%d body=%s", activate.Code, activate.Body.String())
	}

	config := httptest.NewRecorder()
	router.ServeHTTP(config, desktopRequest(http.MethodGet, "/desktop/remote/profiles/"+profile.ID+"/auth/config", nil))
	if config.Code != http.StatusOK || !bytes.Contains(config.Body.Bytes(), []byte(`"registration_mode":"open"`)) {
		t.Fatalf("auth config status=%d body=%s", config.Code, config.Body.String())
	}

	register := httptest.NewRecorder()
	router.ServeHTTP(register, desktopRequest(http.MethodPost, "/desktop/remote/profiles/"+profile.ID+"/register", []byte(`{"email":"u2@example.com","password":"secret123","username":"u2"}`)))
	if register.Code != http.StatusOK || !bytes.Contains(register.Body.Bytes(), []byte(`"success":true`)) {
		t.Fatalf("register status=%d body=%s", register.Code, register.Body.String())
	}

	login := httptest.NewRecorder()
	router.ServeHTTP(login, desktopRequest(http.MethodPost, "/desktop/remote/profiles/"+profile.ID+"/login", []byte(`{"email":"u1@example.com","password":"secret"}`)))
	if login.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}
	if bytes.Contains(login.Body.Bytes(), []byte("access-token")) || bytes.Contains(login.Body.Bytes(), []byte("refresh-token")) {
		t.Fatalf("login leaked token: %s", login.Body.String())
	}

	session := httptest.NewRecorder()
	router.ServeHTTP(session, desktopRequest(http.MethodGet, "/desktop/remote/profiles/"+profile.ID+"/session", nil))
	if session.Code != http.StatusOK || !bytes.Contains(session.Body.Bytes(), []byte(`"authenticated":true`)) {
		t.Fatalf("session status=%d body=%s", session.Code, session.Body.String())
	}

	proxy := httptest.NewRecorder()
	router.ServeHTTP(proxy, desktopRequest(http.MethodGet, "/desktop/remote/profiles/"+profile.ID+"/api/v1/agents", nil))
	if proxy.Code != http.StatusOK || proxy.Body.String() != `{"agents":[]}` {
		t.Fatalf("proxy status=%d body=%s", proxy.Code, proxy.Body.String())
	}

	deleteProfile := httptest.NewRecorder()
	router.ServeHTTP(deleteProfile, desktopRequest(http.MethodDelete, "/desktop/remote/profiles/"+profile.ID, nil))
	if deleteProfile.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", deleteProfile.Code, deleteProfile.Body.String())
	}

	list := httptest.NewRecorder()
	router.ServeHTTP(list, desktopRequest(http.MethodGet, "/desktop/remote/profiles", nil))
	if list.Code != http.StatusOK || list.Body.String() != `[]` {
		t.Fatalf("list after delete status=%d body=%s", list.Code, list.Body.String())
	}
}

func newDesktopRemoteHandlerForTest(t *testing.T) *DesktopRemoteHandler {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	profiles := desktopremote.NewProfileStore(db)
	if err := profiles.AutoMigrate(); err != nil {
		t.Fatal(err)
	}
	sessions := desktopremote.NewSessionStore()
	credentials := newFakeDesktopCredentialStore()
	auth := desktopremote.NewAuthClient(profiles, credentials, sessions, http.DefaultClient)
	manager := desktopremote.NewManager(profiles, auth, http.DefaultClient)
	gateway := desktopremote.NewGatewayWithAuth(profiles, sessions, auth, http.DefaultClient)
	return NewDesktopRemoteHandlerWithSecret(profiles, manager, auth, gateway, sessions, "desktop-secret")
}

func desktopRequest(method, target string, body []byte) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	req.Header.Set("X-Xelora-Desktop-Session", "desktop-secret")
	req.Header.Set("Content-Type", "application/json")
	return req
}

func strconvQuote(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

type fakeDesktopCredentialStore struct {
	values map[string]string
}

func newFakeDesktopCredentialStore() *fakeDesktopCredentialStore {
	return &fakeDesktopCredentialStore{values: map[string]string{}}
}

func (s *fakeDesktopCredentialStore) PutRefreshToken(_ context.Context, profileID, userID, token string) error {
	s.values[profileID+"/"+userID] = token
	return nil
}

func (s *fakeDesktopCredentialStore) GetRefreshToken(_ context.Context, profileID, userID string) (string, error) {
	token, ok := s.values[profileID+"/"+userID]
	if !ok {
		return "", desktopremote.ErrCredentialNotFound
	}
	return token, nil
}

func (s *fakeDesktopCredentialStore) DeleteRefreshToken(_ context.Context, profileID, userID string) error {
	delete(s.values, profileID+"/"+userID)
	return nil
}
