package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/Xelora/internal/config"
	apperrors "github.com/Tencent/Xelora/internal/errors"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// stubRegisterUserService is a UserService whose ONLY useful method is
// Register; every other call panics. Using an interface embedding plus a
// targeted override keeps the test focused on the Register handler's
// branching logic without dragging in the entire user service surface.
type stubRegisterUserService struct {
	interfaces.UserService
	register       func(ctx context.Context, req *types.RegisterRequest) (*types.User, error)
	getUserByEmail func(ctx context.Context, email string) (*types.User, error)
	generateTokens func(ctx context.Context, user *types.User) (string, string, error)
}

func (s *stubRegisterUserService) Register(ctx context.Context, req *types.RegisterRequest) (*types.User, error) {
	return s.register(ctx, req)
}

func (s *stubRegisterUserService) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	return s.getUserByEmail(ctx, email)
}

func (s *stubRegisterUserService) GenerateTokens(ctx context.Context, user *types.User) (string, string, error) {
	return s.generateTokens(ctx, user)
}

type stubAutoSetupTenantService struct {
	interfaces.TenantService
	getTenantByID func(ctx context.Context, id uint64) (*types.Tenant, error)
}

func (s *stubAutoSetupTenantService) GetTenantByID(ctx context.Context, id uint64) (*types.Tenant, error) {
	return s.getTenantByID(ctx, id)
}

// errorCapture mirrors gin's default ErrorHandler behaviour for tests:
// when a handler calls c.Error(), we surface it as an HTTP response so the
// recorder reflects the real client-visible status. The production
// middleware does the same thing in middleware/error_handler.go.
func errorCapture() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		err := c.Errors.Last().Err
		if appErr, ok := err.(*apperrors.AppError); ok {
			c.JSON(appErr.HTTPCode, gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func newRegisterTestRouter(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(errorCapture())
	r.POST("/auth/register", h.Register)
	return r
}

func newAutoSetupTestRouter(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(errorCapture())
	r.POST("/auth/auto-setup", h.AutoSetup)
	return r
}

func doRegister(t *testing.T, r *gin.Engine, body any) *httptest.ResponseRecorder {
	t.Helper()
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// validRegisterBody returns a payload that passes parameter validation, so
// each test is exercising the gate logic and not the body parser.
func validRegisterBody() map[string]string {
	return map[string]string{
		"username": "alice",
		"email":    "alice@example.com",
		"password": "supersecret",
	}
}

func TestRegister_InviteOnlyRejects(t *testing.T) {
	// PR 3 (#1303): when auth.registration_mode=invite_only, Register
	// must respond 403 BEFORE touching the user service. The frontend
	// already hides the sign-up link via /auth/config; this is the
	// server-side enforcement for direct API hits.
	called := false
	us := &stubRegisterUserService{
		register: func(context.Context, *types.RegisterRequest) (*types.User, error) {
			called = true
			return &types.User{ID: "u1"}, nil
		},
	}
	h := NewAuthHandler(&config.Config{
		Auth: &config.AuthConfig{RegistrationMode: config.AuthRegistrationModeInviteOnly},
	}, us, nil, nil, nil)

	w := doRegister(t, newRegisterTestRouter(h), validRegisterBody())
	if w.Code != http.StatusForbidden {
		t.Fatalf("invite_only must return 403, got %d body=%s", w.Code, w.Body.String())
	}
	if called {
		t.Fatalf("UserService.Register must not be called when invite_only blocks the request")
	}
}

func TestRegister_SelfServeAllowsRegistration(t *testing.T) {
	// Default registration_mode keeps PR 1 behaviour intact: the gate
	// is dormant and the request reaches the user service. We don't
	// exercise the real service here — just confirm the gate let it
	// through by observing the stub being invoked.
	called := false
	us := &stubRegisterUserService{
		register: func(_ context.Context, _ *types.RegisterRequest) (*types.User, error) {
			called = true
			return &types.User{ID: "u1", Email: "alice@example.com"}, nil
		},
	}
	h := NewAuthHandler(&config.Config{
		Auth: &config.AuthConfig{RegistrationMode: config.AuthRegistrationModeSelfServe},
	}, us, nil, nil, nil)

	w := doRegister(t, newRegisterTestRouter(h), validRegisterBody())
	if w.Code != http.StatusCreated {
		t.Fatalf("self_serve must allow registration, got %d body=%s", w.Code, w.Body.String())
	}
	if !called {
		t.Fatalf("UserService.Register should have been invoked")
	}
}

func TestRegister_NilAuthConfigDoesNotPanic(t *testing.T) {
	// Defensive: a nil Auth section means the operator hasn't set the
	// registration mode at all, which must not crash and must keep the
	// legacy "registration enabled" behaviour. Mirrors the nil guard in
	// the handler so a config-loading bug doesn't take the server down.
	us := &stubRegisterUserService{
		register: func(_ context.Context, _ *types.RegisterRequest) (*types.User, error) {
			return &types.User{ID: "u1", Email: "alice@example.com"}, nil
		},
	}
	h := NewAuthHandler(&config.Config{}, us, nil, nil, nil)

	w := doRegister(t, newRegisterTestRouter(h), validRegisterBody())
	if w.Code != http.StatusCreated {
		t.Fatalf("nil Auth config must fall back to allow, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAutoSetup_PersonalEditionAllowed(t *testing.T) {
	previousEdition := Edition
	Edition = "personal"
	t.Cleanup(func() {
		Edition = previousEdition
	})

	defaultUser := &types.User{
		ID:       "u1",
		Username: "personal",
		Email:    "admin@xelora.local",
		TenantID: 42,
		IsActive: true,
	}
	us := &stubRegisterUserService{
		getUserByEmail: func(_ context.Context, email string) (*types.User, error) {
			if email != "admin@xelora.local" {
				t.Fatalf("unexpected auto-setup email: %s", email)
			}
			return defaultUser, nil
		},
		generateTokens: func(_ context.Context, user *types.User) (string, string, error) {
			if user != defaultUser {
				t.Fatalf("GenerateTokens got unexpected user: %#v", user)
			}
			return "access-token", "refresh-token", nil
		},
	}
	ts := &stubAutoSetupTenantService{
		getTenantByID: func(_ context.Context, id uint64) (*types.Tenant, error) {
			if id != 42 {
				t.Fatalf("unexpected tenant id: %d", id)
			}
			return &types.Tenant{ID: id, Name: "Personal"}, nil
		},
	}
	h := NewAuthHandler(&config.Config{}, us, ts, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/auth/auto-setup", nil)
	w := httptest.NewRecorder()
	newAutoSetupTestRouter(h).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("personal edition auto-setup must be allowed, got %d body=%s", w.Code, w.Body.String())
	}
	var body types.LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode auto-setup response: %v body=%s", err, w.Body.String())
	}
	if !body.Success || body.Token != "access-token" || body.RefreshToken != "refresh-token" {
		t.Fatalf("unexpected auto-setup response: %+v", body)
	}
	if len(body.Memberships) != 1 || body.Memberships[0].TenantName != "Personal" {
		t.Fatalf("unexpected memberships: %+v", body.Memberships)
	}
}
