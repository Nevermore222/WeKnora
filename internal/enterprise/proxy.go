package enterprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const proxyTimeout = 120 * time.Second

// Proxy forwards requests to an enterprise server on behalf of the local client.
// It authenticates as the linked server user (JWT) when available, falling back
// to the tenant-scoped API token, and streams responses back.
type Proxy struct {
	client *http.Client
	store  *Store
}

// NewProxy creates a Proxy with a sensible timeout for chat/retrieval operations.
// The store is used to persist refreshed JWTs.
func NewProxy(store *Store) *Proxy {
	return &Proxy{
		client: &http.Client{Timeout: proxyTimeout},
		store:  store,
	}
}

// applyAuth sets the authentication header on the outgoing request. It prefers
// the linked user JWT (so the server sees the specific provisioned user) and
// falls back to the API token (tenant-scoped) when not linked.
func applyAuth(req *http.Request, conn *ServerConnection) {
	if conn.Config.ServerJWT != "" {
		req.Header.Set("Authorization", "Bearer "+conn.Config.ServerJWT)
	} else if conn.Config.APIToken != "" {
		req.Header.Set("X-API-Key", conn.Config.APIToken)
	}
}

// ForwardRequest proxies a single HTTP request to the enterprise server and
// returns the upstream response for the caller to stream back to the client.
func (p *Proxy) ForwardRequest(ctx context.Context, conn *ServerConnection, method, path string, body io.Reader, headers http.Header) (*http.Response, error) {
	if conn == nil || conn.Status != StatusConnected {
		return nil, fmt.Errorf("enterprise server is not connected")
	}

	base := strings.TrimRight(conn.Config.BaseURL, "/")
	url := base + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("build proxy request: %w", err)
	}

	applyAuth(req, conn)

	// Forward select headers from the original request.
	for _, key := range []string{"Content-Type", "Accept", "X-Tenant-ID"} {
		if v := headers.Get(key); v != "" {
			req.Header.Set(key, v)
		}
	}

	resp, err := p.client.Do(req)
	if err != nil {
		conn.Status = StatusError
		conn.LastError = err.Error()
		return nil, fmt.Errorf("proxy request failed: %w", err)
	}

	return resp, nil
}

// ForwardJSON proxies a JSON request/response. If the server returns 401 and a
// refresh token is available, it refreshes the JWT (persisting it) and retries
// the request once.
func (p *Proxy) ForwardJSON(ctx context.Context, conn *ServerConnection, method, path string, jsonBody []byte, headers http.Header) (*http.Response, error) {
	if headers == nil {
		headers = http.Header{}
	}
	headers.Set("Content-Type", "application/json")

	newBody := func() io.Reader {
		if len(jsonBody) == 0 {
			return nil
		}
		return bytes.NewReader(jsonBody)
	}

	resp, err := p.ForwardRequest(ctx, conn, method, path, newBody(), headers)
	if err != nil {
		return nil, err
	}

	// On 401, try a one-time token refresh + retry.
	if resp.StatusCode == http.StatusUnauthorized && conn.Config.ServerRefreshToken != "" {
		_ = resp.Body.Close()
		if rerr := p.refreshJWT(ctx, conn); rerr == nil {
			return p.ForwardRequest(ctx, conn, method, path, newBody(), headers)
		}
		// Refresh failed — re-issue the original request so the caller sees the
		// 401 and can trigger re-provisioning at a higher level.
		return p.ForwardRequest(ctx, conn, method, path, newBody(), headers)
	}

	return resp, nil
}

// ForwardSSE proxies a streaming SSE request (used for chat). The caller is
// responsible for copying resp.Body to the client's response writer.
func (p *Proxy) ForwardSSE(ctx context.Context, conn *ServerConnection, path string, jsonBody []byte, headers http.Header) (*http.Response, error) {
	if headers == nil {
		headers = http.Header{}
	}
	headers.Set("Accept", "text/event-stream")
	return p.ForwardJSON(ctx, conn, http.MethodPost, path, jsonBody, headers)
}

// refreshJWT exchanges the stored refresh token for a new access token and
// persists it (both in-memory and, when a store is available, at rest).
func (p *Proxy) refreshJWT(ctx context.Context, conn *ServerConnection) error {
	base := strings.TrimRight(conn.Config.BaseURL, "/")
	payload, _ := json.Marshal(map[string]string{"refreshToken": conn.Config.ServerRefreshToken})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/auth/refresh", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("refresh HTTP %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return err
	}
	if parsed.AccessToken == "" {
		return fmt.Errorf("refresh returned no access token")
	}
	conn.Config.ServerJWT = parsed.AccessToken
	if parsed.RefreshToken != "" {
		conn.Config.ServerRefreshToken = parsed.RefreshToken
	}
	if p.store != nil {
		_ = p.store.SaveLinkedIdentity(ctx, conn.Config.ID,
			conn.Config.LinkedUserID, conn.Config.LinkedEmail, conn.Config.LinkedTenantID,
			conn.Config.LinkedPassword, conn.Config.ServerJWT, conn.Config.ServerRefreshToken)
	}
	return nil
}
