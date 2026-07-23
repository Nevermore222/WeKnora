package main

import (
	"context"
	"strings"
)

// App holds Wails-bound state for the desktop shell.
type App struct {
	ctx                  context.Context
	backendURL           string
	apiLanBaseURL        string
	desktopSessionSecret string
	listenPublic         bool
	shutdownCh           chan struct{}
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{
		shutdownCh: make(chan struct{}, 1),
	}
}

// startup is called when the application starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
	a.shutdownCh <- struct{}{}
}

// GetAPIBaseURL returns the local HTTP base URL for REST API calls (e.g. http://127.0.0.1:PORT/api/v1).
// The desktop shell proxies the webview to this address; window.location.origin is not the API host.
func (a *App) GetAPIBaseURL() string {
	if a.backendURL == "" {
		return ""
	}
	return strings.TrimRight(a.backendURL, "/") + "/api/v1"
}

type DesktopBootstrap struct {
	APIBaseURL                      string `json:"api_base_url"`
	Session                         string `json:"session"`
	DefaultEnterpriseServerURL      string `json:"default_enterprise_server_url,omitempty"`
	DefaultEnterpriseServerName     string `json:"default_enterprise_server_name,omitempty"`
	DefaultEnterpriseServerRequired bool   `json:"default_enterprise_server_required"`
	DefaultEnterpriseAllowInsecure  bool   `json:"default_enterprise_allow_insecure"`
}

func (a *App) GetDesktopBootstrap() DesktopBootstrap {
	return DesktopBootstrap{
		APIBaseURL:                      strings.TrimRight(a.backendURL, "/"),
		Session:                         a.desktopSessionSecret,
		DefaultEnterpriseServerURL:      defaultEnterpriseServerURL(),
		DefaultEnterpriseServerName:     defaultEnterpriseServerName(),
		DefaultEnterpriseServerRequired: defaultEnterpriseServerURL() != "",
		DefaultEnterpriseAllowInsecure:  defaultEnterpriseAllowInsecure(),
	}
}

// GetDesktopHTTPPortSetting returns the saved local API port (0 = random port each launch).
func (a *App) GetDesktopHTTPPortSetting() int {
	return LoadDesktopPrefsHTTPPort()
}

// SetDesktopHTTPPortSetting saves the preferred local API port to application support. Restart the app for it to take effect unless it matches the current listener.
func (a *App) SetDesktopHTTPPortSetting(port int) error {
	return SaveDesktopHTTPPortPreference(port)
}

// GetDesktopHTTPBindPublicSetting returns whether API listens on all interfaces (0.0.0.0).
func (a *App) GetDesktopHTTPBindPublicSetting() bool {
	return LoadDesktopHTTPBindPublic()
}

// SetDesktopHTTPBindPublicSetting saves LAN/public listen preference. Restart the app for it to take effect.
func (a *App) SetDesktopHTTPBindPublicSetting(v bool) error {
	return SaveDesktopHTTPBindPublicPreference(v)
}

// GetDesktopSandboxMode returns the saved skill execution mode ("local" or "docker").
func (a *App) GetDesktopSandboxMode() string {
	return LoadDesktopSandboxMode()
}

// SetDesktopSandboxMode saves the skill execution mode and applies it immediately
// by updating the XELORA_SANDBOX_MODE environment variable.
func (a *App) SetDesktopSandboxMode(mode string) error {
	return SaveDesktopSandboxModePreference(mode)
}

// GetAPILanBaseURL returns a suggested base URL for other devices on the LAN (…/api/v1), or empty if not in bind-public mode or IP detection failed.
func (a *App) GetAPILanBaseURL() string {
	return a.apiLanBaseURL
}

// GetDesktopListenPublicActive is true when this session’s API server is listening on all interfaces (runtime), not the saved preference.
func (a *App) GetDesktopListenPublicActive() bool {
	return a.listenPublic
}

// CheckForUpdates manually triggers the update check from the frontend.
func (a *App) CheckForUpdates() {
	if a.ctx != nil {
		checkUpdate(a.ctx, desktopAboutVersion(), true, false)
	}
}

// AutoCheckForUpdates silently checks for updates and downloads them.
func (a *App) AutoCheckForUpdates() {
	if a.ctx != nil {
		checkUpdate(a.ctx, desktopAboutVersion(), false, true)
	}
}
