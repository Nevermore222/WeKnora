package desktopremote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const gatewayPathPrefix = "/api/v1/desktop/enterprise/"

var forwardedHeaders = map[string]bool{
	"Accept":          true,
	"Accept-Language": true,
	"Content-Type":    true,
	"Content-Range":   true,
	"Range":           true,
	"X-Request-Id":    true,
	"X-Tenant-Id":     true,
	"Idempotency-Key": true,
}

var responseHeaders = map[string]bool{
	"Accept-Ranges":          true,
	"Content-Disposition":    true,
	"Content-Range":          true,
	"Content-Type":           true,
	"Etag":                   true,
	"Last-Modified":          true,
	"X-Accel-Buffering":      true,
	"X-Content-Type-Options": true,
	"X-Request-Id":           true,
}

type Gateway struct {
	profiles   ProfileSource
	sessions   *SessionStore
	auth       *AuthClient
	httpClient *http.Client
}

func NewGateway(profiles ProfileSource, sessions *SessionStore, httpClient *http.Client) *Gateway {
	if sessions == nil {
		sessions = NewSessionStore()
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	client := *httpClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Gateway{profiles: profiles, sessions: sessions, httpClient: &client}
}

func NewGatewayWithAuth(profiles ProfileSource, sessions *SessionStore, auth *AuthClient, httpClient *http.Client) *Gateway {
	gateway := NewGateway(profiles, sessions, httpClient)
	gateway.auth = auth
	return gateway
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	profileID, remotePath, err := parseGatewayPath(r.URL.Path)
	if err != nil {
		writeGatewayError(w, http.StatusBadRequest, "invalid_remote_path")
		return
	}

	profile, err := g.profile(r.Context(), profileID)
	if err != nil {
		writeGatewayError(w, http.StatusNotFound, "server_profile_not_found")
		return
	}
	target, err := profile.ResolveTarget(remotePath)
	if err != nil {
		writeGatewayError(w, http.StatusBadRequest, "invalid_remote_target")
		return
	}
	target.RawQuery = r.URL.RawQuery

	session, ok := g.sessions.Get(profileID)
	if !ok || strings.TrimSpace(session.AccessToken) == "" {
		writeGatewayError(w, http.StatusUnauthorized, "desktop_session_required")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeGatewayError(w, http.StatusBadRequest, "invalid_remote_request")
		return
	}

	resp, err := g.forward(r, target.String(), body, session.AccessToken)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		writeGatewayError(w, http.StatusBadGateway, "remote_request_failed")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized && g.auth != nil && replayable(r) {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		accessToken, refreshErr := g.auth.RefreshAccessToken(r.Context(), profileID)
		if refreshErr != nil {
			writeGatewayError(w, http.StatusUnauthorized, "desktop_session_refresh_failed")
			return
		}
		resp, err = g.forward(r, target.String(), body, accessToken)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			writeGatewayError(w, http.StatusBadGateway, "remote_request_failed")
			return
		}
		defer resp.Body.Close()
	}

	copyGatewayResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	streamGatewayBody(w, resp.Body)
}

func (g *Gateway) forward(original *http.Request, target string, body []byte, accessToken string) (*http.Response, error) {
	upstreamReq, err := http.NewRequestWithContext(original.Context(), original.Method, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	upstreamReq.ContentLength = int64(len(body))
	if original.ContentLength < 0 {
		upstreamReq.ContentLength = -1
	}
	copyGatewayRequestHeaders(upstreamReq.Header, original.Header)
	upstreamReq.Header.Set("Authorization", "Bearer "+accessToken)
	return g.httpClient.Do(upstreamReq)
}

func (g *Gateway) profile(ctx context.Context, profileID string) (*ServerProfile, error) {
	if g.profiles == nil {
		return nil, errors.New("profile source is required")
	}
	return g.profiles.Get(ctx, profileID)
}

func parseGatewayPath(rawPath string) (string, string, error) {
	if !strings.HasPrefix(rawPath, gatewayPathPrefix) {
		return "", "", fmt.Errorf("gateway path must start with %s", gatewayPathPrefix)
	}
	rest := strings.TrimPrefix(rawPath, gatewayPathPrefix)
	proxyIndex := strings.Index(rest, "/proxy")
	if proxyIndex <= 0 {
		return "", "", errors.New("gateway path missing profile or proxy marker")
	}
	profileID := rest[:proxyIndex]
	remotePath := strings.TrimPrefix(rest[proxyIndex:], "/proxy")
	if strings.TrimSpace(profileID) == "" {
		return "", "", errors.New("profile id is required")
	}
	if remotePath == "" {
		remotePath = "/"
	}
	if !strings.HasPrefix(remotePath, "/") {
		remotePath = "/" + remotePath
	}
	return profileID, remotePath, nil
}

func copyGatewayRequestHeaders(dst, src http.Header) {
	for name, values := range src {
		canonical := http.CanonicalHeaderKey(name)
		if !forwardedHeaders[canonical] {
			continue
		}
		for _, value := range values {
			dst.Add(canonical, value)
		}
	}
}

func copyGatewayResponseHeaders(dst, src http.Header) {
	for name, values := range src {
		canonical := http.CanonicalHeaderKey(name)
		if !responseHeaders[canonical] {
			continue
		}
		for _, value := range values {
			dst.Add(canonical, value)
		}
	}
}

func replayable(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return r.Header.Get("Idempotency-Key") != ""
	}
}

func streamGatewayBody(w http.ResponseWriter, body io.Reader) {
	buf := make([]byte, 32*1024)
	flusher, _ := w.(http.Flusher)
	for {
		n, readErr := body.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
			if flusher != nil {
				flusher.Flush()
			}
		}
		if readErr != nil {
			return
		}
	}
}

func writeGatewayError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}
