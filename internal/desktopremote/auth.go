package desktopremote

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ProfileSource interface {
	Get(ctx context.Context, id string) (*ServerProfile, error)
}

type AuthClient struct {
	profiles    ProfileSource
	credentials CredentialStore
	sessions    *SessionStore
	httpClient  *http.Client
	now         func() time.Time
}

func NewAuthClient(profiles ProfileSource, credentials CredentialStore, sessions *SessionStore, httpClient *http.Client) *AuthClient {
	if sessions == nil {
		sessions = NewSessionStore()
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &AuthClient{
		profiles:    profiles,
		credentials: credentials,
		sessions:    sessions,
		httpClient:  httpClient,
		now:         time.Now,
	}
}

func (c *AuthClient) Login(ctx context.Context, profileID string, body io.Reader) (IdentitySnapshot, error) {
	profile, err := c.profile(ctx, profileID)
	if err != nil {
		return IdentitySnapshot{}, err
	}
	target, err := profile.ResolveTarget("/api/v1/auth/login")
	if err != nil {
		return IdentitySnapshot{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), body)
	if err != nil {
		return IdentitySnapshot{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	var response authTokenResponse
	if err := c.doJSON(req, &response); err != nil {
		return IdentitySnapshot{}, err
	}
	if !response.Success {
		return IdentitySnapshot{}, errors.New("login failed")
	}
	accessToken := strings.TrimSpace(firstNonEmpty(response.Token, response.AccessToken))
	refreshToken := strings.TrimSpace(response.RefreshToken)
	if accessToken == "" || refreshToken == "" {
		return IdentitySnapshot{}, errors.New("login response missing token")
	}

	snapshot := response.snapshot(c.now(), accessToken)
	if snapshot.UserID == "" {
		return IdentitySnapshot{}, errors.New("login response missing user id")
	}
	if err := c.credentials.PutRefreshToken(ctx, profileID, snapshot.UserID, refreshToken); err != nil {
		return IdentitySnapshot{}, err
	}

	c.sessions.Set(profileID, &Session{
		UserID:            snapshot.UserID,
		AccessToken:       accessToken,
		RefreshTokenOwner: snapshot.UserID,
		AccessExpiresAt:   snapshot.ExpiresAt,
		Snapshot:          snapshot,
	})
	return snapshot, nil
}

func (c *AuthClient) AccessToken(ctx context.Context, profileID string) (string, error) {
	session, ok := c.sessions.Get(profileID)
	if !ok {
		return "", ErrCredentialNotFound
	}
	if session.AccessToken != "" && session.AccessExpiresAt.After(c.now().Add(30*time.Second)) {
		return session.AccessToken, nil
	}
	return c.refresh(ctx, profileID, session)
}

func (c *AuthClient) RefreshAccessToken(ctx context.Context, profileID string) (string, error) {
	session, ok := c.sessions.Get(profileID)
	if !ok {
		return "", ErrCredentialNotFound
	}
	return c.refresh(ctx, profileID, session)
}

func (c *AuthClient) Logout(ctx context.Context, profileID string) error {
	session, ok := c.sessions.Get(profileID)
	c.sessions.Delete(profileID)
	if !ok || session.RefreshTokenOwner == "" {
		return nil
	}
	if err := c.credentials.DeleteRefreshToken(ctx, profileID, session.RefreshTokenOwner); err != nil && !errors.Is(err, ErrCredentialNotFound) {
		return err
	}
	return nil
}

func (c *AuthClient) refresh(ctx context.Context, profileID string, session *Session) (string, error) {
	if session == nil || session.RefreshTokenOwner == "" {
		return "", ErrCredentialNotFound
	}
	refreshToken, err := c.credentials.GetRefreshToken(ctx, profileID, session.RefreshTokenOwner)
	if err != nil {
		return "", err
	}

	profile, err := c.profile(ctx, profileID)
	if err != nil {
		return "", err
	}
	target, err := profile.ResolveTarget("/api/v1/auth/refresh")
	if err != nil {
		return "", err
	}

	body, err := json.Marshal(map[string]string{"refreshToken": refreshToken})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	var response authTokenResponse
	if err := c.doJSON(req, &response); err != nil {
		return "", err
	}
	accessToken := strings.TrimSpace(firstNonEmpty(response.AccessToken, response.Token))
	newRefreshToken := strings.TrimSpace(response.RefreshToken)
	if accessToken == "" || newRefreshToken == "" {
		return "", errors.New("refresh response missing token")
	}
	if err := c.credentials.PutRefreshToken(ctx, profileID, session.RefreshTokenOwner, newRefreshToken); err != nil {
		return "", err
	}

	session.AccessToken = accessToken
	session.AccessExpiresAt = tokenExpiry(accessToken, c.now())
	session.Snapshot.ExpiresAt = session.AccessExpiresAt
	c.sessions.Set(profileID, session)
	return accessToken, nil
}

func (c *AuthClient) profile(ctx context.Context, profileID string) (*ServerProfile, error) {
	if c.profiles == nil {
		return nil, errors.New("profile source is required")
	}
	return c.profiles.Get(ctx, profileID)
}

func (c *AuthClient) doJSON(req *http.Request, target any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("remote auth failed: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return err
	}
	return nil
}

type authTokenResponse struct {
	Success      bool              `json:"success"`
	User         json.RawMessage   `json:"user"`
	ActiveTenant json.RawMessage   `json:"active_tenant"`
	Tenant       json.RawMessage   `json:"tenant"`
	Memberships  []json.RawMessage `json:"memberships"`
	Token        string            `json:"token"`
	AccessToken  string            `json:"access_token"`
	RefreshToken string            `json:"refresh_token"`
}

func (r authTokenResponse) snapshot(now time.Time, accessToken string) IdentitySnapshot {
	tenant := r.ActiveTenant
	if len(tenant) == 0 {
		tenant = r.Tenant
	}
	return IdentitySnapshot{
		Authenticated: true,
		UserID:        stringIDFromRaw(r.User, "id"),
		TenantID:      uintIDFromRaw(tenant, "id"),
		User:          cloneRawMessage(r.User),
		Tenant:        cloneRawMessage(tenant),
		Memberships:   cloneRawMessages(r.Memberships),
		ExpiresAt:     tokenExpiry(accessToken, now),
	}
}

func cloneRawMessages(values []json.RawMessage) []json.RawMessage {
	if values == nil {
		return nil
	}
	cloned := make([]json.RawMessage, len(values))
	for i := range values {
		cloned[i] = cloneRawMessage(values[i])
	}
	return cloned
}

func stringIDFromRaw(raw json.RawMessage, key string) string {
	var values map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &values) != nil {
		return ""
	}
	switch value := values[key].(type) {
	case string:
		return value
	case float64:
		return strconv.FormatUint(uint64(value), 10)
	default:
		return ""
	}
}

func uintIDFromRaw(raw json.RawMessage, key string) uint64 {
	var values map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &values) != nil {
		return 0
	}
	switch value := values[key].(type) {
	case float64:
		return uint64(value)
	case string:
		parsed, _ := strconv.ParseUint(value, 10, 64)
		return parsed
	default:
		return 0
	}
}

func tokenExpiry(token string, now time.Time) time.Time {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return now.Add(15 * time.Minute)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return now.Add(15 * time.Minute)
	}
	var claims struct {
		ExpiresAt int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.ExpiresAt <= 0 {
		return now.Add(15 * time.Minute)
	}
	return time.Unix(claims.ExpiresAt, 0)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
