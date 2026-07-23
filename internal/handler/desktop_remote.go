package handler

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"io"
	"net/http"
	"strings"

	"github.com/Tencent/Xelora/internal/desktopremote"
	"github.com/gin-gonic/gin"
)

const DesktopSessionHeader = "X-Xelora-Desktop-Session"

type DesktopRemoteHandler struct {
	profiles *desktopremote.ProfileStore
	manager  *desktopremote.Manager
	auth     *desktopremote.AuthClient
	gateway  *desktopremote.Gateway
	sessions *desktopremote.SessionStore
	secret   string
}

func NewDesktopRemoteHandler(
	profiles *desktopremote.ProfileStore,
	manager *desktopremote.Manager,
	auth *desktopremote.AuthClient,
	gateway *desktopremote.Gateway,
	sessions *desktopremote.SessionStore,
) *DesktopRemoteHandler {
	return NewDesktopRemoteHandlerWithSecret(profiles, manager, auth, gateway, sessions, newDesktopSessionSecret())
}

func NewDesktopRemoteHandlerWithSecret(
	profiles *desktopremote.ProfileStore,
	manager *desktopremote.Manager,
	auth *desktopremote.AuthClient,
	gateway *desktopremote.Gateway,
	sessions *desktopremote.SessionStore,
	secret string,
) *DesktopRemoteHandler {
	return &DesktopRemoteHandler{
		profiles: profiles,
		manager:  manager,
		auth:     auth,
		gateway:  gateway,
		sessions: sessions,
		secret:   strings.TrimSpace(secret),
	}
}

func (h *DesktopRemoteHandler) SessionSecret() string {
	return h.secret
}

func (h *DesktopRemoteHandler) RequireDesktopSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		got := []byte(c.GetHeader(DesktopSessionHeader))
		want := []byte(h.secret)
		if len(got) != len(want) || subtle.ConstantTimeCompare(got, want) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "desktop_session_required"})
			return
		}
		c.Next()
	}
}

func (h *DesktopRemoteHandler) RegisterRoutes(r gin.IRouter) {
	r.GET("/profiles", h.ListProfiles)
	r.POST("/profiles", h.CreateProfile)
	r.POST("/profiles/:id/activate", h.ActivateProfile)
	r.POST("/profiles/:id/login", h.Login)
	r.GET("/profiles/:id/auth/config", h.AuthConfig)
	r.POST("/profiles/:id/register", h.Register)
	r.POST("/profiles/:id/logout", h.Logout)
	r.GET("/profiles/:id/session", h.Session)
	r.DELETE("/profiles/:id", h.DeleteProfile)
	r.Any("/profiles/:id/api/v1/*path", h.ProxyAPI)
}

func (h *DesktopRemoteHandler) ListProfiles(c *gin.Context) {
	profiles, err := h.profiles.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profiles)
}

func (h *DesktopRemoteHandler) CreateProfile(c *gin.Context) {
	var req struct {
		Name                          string `json:"name"`
		BaseURL                       string `json:"base_url"`
		AllowInsecureTransport        bool   `json:"allow_insecure_transport"`
		TrustedCertificateFingerprint string `json:"trusted_certificate_fingerprint"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_profile"})
		return
	}
	profile, err := h.profiles.Create(c.Request.Context(), desktopremote.ServerProfile{
		Name:                          req.Name,
		BaseURL:                       req.BaseURL,
		AllowInsecureTransport:        req.AllowInsecureTransport,
		TrustedCertificateFingerprint: req.TrustedCertificateFingerprint,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, profile)
}

func (h *DesktopRemoteHandler) ActivateProfile(c *gin.Context) {
	capabilities, err := h.manager.Activate(c.Request.Context(), c.Param("id"))
	if err != nil {
		status := http.StatusBadGateway
		if err == desktopremote.ErrIncompatibleAPIContract {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, capabilities)
}

func (h *DesktopRemoteHandler) Login(c *gin.Context) {
	snapshot, err := h.auth.Login(c.Request.Context(), c.Param("id"), c.Request.Body)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, snapshot)
}

func (h *DesktopRemoteHandler) AuthConfig(c *gin.Context) {
	h.forwardAnonymousAuth(c, http.MethodGet, "/api/v1/auth/config")
}

func (h *DesktopRemoteHandler) Register(c *gin.Context) {
	h.forwardAnonymousAuth(c, http.MethodPost, "/api/v1/auth/register")
}

func (h *DesktopRemoteHandler) Logout(c *gin.Context) {
	if err := h.auth.Logout(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *DesktopRemoteHandler) DeleteProfile(c *gin.Context) {
	profileID := c.Param("id")
	if err := h.auth.Logout(c.Request.Context(), profileID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.profiles.Delete(c.Request.Context(), profileID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *DesktopRemoteHandler) Session(c *gin.Context) {
	c.JSON(http.StatusOK, h.sessions.Snapshot(c.Param("id")))
}

func (h *DesktopRemoteHandler) ProxyAPI(c *gin.Context) {
	profileID := c.Param("id")
	remotePath := "/api/v1" + c.Param("path")
	c.Request.URL.Path = "/api/v1/desktop/enterprise/" + profileID + "/proxy" + remotePath
	h.gateway.ServeHTTP(c.Writer, c.Request)
}

func (h *DesktopRemoteHandler) forwardAnonymousAuth(c *gin.Context, method, remotePath string) {
	profile, err := h.profiles.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server_profile_not_found"})
		return
	}
	target, err := profile.ResolveTarget(remotePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_remote_target"})
		return
	}
	target.RawQuery = c.Request.URL.RawQuery
	req, err := http.NewRequestWithContext(c.Request.Context(), method, target.String(), c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_remote_request"})
		return
	}
	req.Header.Set("Accept", "application/json")
	if contentType := c.GetHeader("Content-Type"); contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if language := c.GetHeader("Accept-Language"); language != "" {
		req.Header.Set("Accept-Language", language)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "remote_request_failed"})
		return
	}
	defer resp.Body.Close()

	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		c.Header("Content-Type", contentType)
	}
	c.Status(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}

func newDesktopSessionSecret() string {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw[:])
}
