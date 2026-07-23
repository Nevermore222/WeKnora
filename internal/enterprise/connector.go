package enterprise

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Tencent/Xelora/internal/logger"
)

const (
	defaultHealthInterval = 30 * time.Second
	defaultRequestTimeout = 30 * time.Second
	defaultConnectTimeout = 10 * time.Second
	// Exponential backoff for auto-reconnect after a connection drops.
	initialBackoff = 5 * time.Second
	maxBackoff     = 5 * time.Minute
	backoffFactor  = 2.0
)

// ServerConnection represents an active connection to an enterprise server.
type ServerConnection struct {
	Config       ServerConfig
	Status       ConnectionStatus
	Capabilities *ServerCapabilities
	LastError    string
	// LastSyncedAt records when capabilities were last successfully fetched,
	// so the UI can show how stale the cached resource list is.
	LastSyncedAt time.Time
	// nextRetryAt and currentBackoff drive exponential-backoff auto-reconnect.
	nextRetryAt    time.Time
	currentBackoff time.Duration
	client         *http.Client
}

// Connector manages the lifecycle of enterprise server connections.
type Connector struct {
	store       *Store
	mu          sync.RWMutex
	connections map[string]*ServerConnection
	healthCh    chan HealthEvent
	stopCh      chan struct{}
	started     bool
}

// NewConnector creates a Connector backed by the given store.
func NewConnector(store *Store) *Connector {
	return &Connector{
		store:       store,
		connections: make(map[string]*ServerConnection),
		healthCh:    make(chan HealthEvent, 32),
		stopCh:      make(chan struct{}),
	}
}

// Start launches the background health check loop and auto-connects servers
// that have AutoConnect enabled.
func (c *Connector) Start(ctx context.Context) {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return
	}
	c.started = true
	c.mu.Unlock()

	// Auto-connect servers marked for it.
	servers, err := c.store.ListServers(ctx)
	if err != nil {
		logger.Warnf(ctx, "enterprise: failed to list servers for auto-connect: %v", err)
	} else {
		for _, srv := range servers {
			if srv.AutoConnect {
				go func(s ServerConfig) {
					_ = c.Connect(context.Background(), s.ID)
				}(srv)
			}
		}
	}

	// Background health loop.
	go c.healthLoop()
}

// Stop shuts down the connector and all active connections.
func (c *Connector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started {
		return
	}
	close(c.stopCh)
	c.started = false
	for id, conn := range c.connections {
		conn.Status = StatusDisconnected
		delete(c.connections, id)
	}
}

// HealthEvents returns the channel on which health status changes are emitted.
func (c *Connector) HealthEvents() <-chan HealthEvent {
	return c.healthCh
}

// Connect establishes a connection to the server with the given ID.
func (c *Connector) Connect(ctx context.Context, serverID string) error {
	srv, err := c.store.GetServer(ctx, serverID)
	if err != nil {
		return fmt.Errorf("enterprise: server not found: %w", err)
	}

	conn := &ServerConnection{
		Config: *srv,
		Status: StatusConnecting,
		client: &http.Client{Timeout: defaultRequestTimeout},
	}

	c.mu.Lock()
	c.connections[serverID] = conn
	c.mu.Unlock()

	c.emitHealth(serverID, StatusConnecting, "")

	// Verify connectivity and discover capabilities.
	caps, err := c.discover(ctx, conn)
	if err != nil {
		conn.Status = StatusError
		conn.LastError = err.Error()
		c.emitHealth(serverID, StatusError, err.Error())
		return fmt.Errorf("enterprise: connect failed: %w", err)
	}

	conn.Status = StatusConnected
	conn.Capabilities = caps
	conn.LastError = ""
	conn.LastSyncedAt = time.Now()
	_ = c.store.TouchServer(ctx, serverID)
	c.emitHealth(serverID, StatusConnected, "")

	// Cache discovered resources for offline UI rendering.
	c.cacheCapabilities(ctx, serverID, caps)

	logger.Infof(ctx, "enterprise: connected to %q (%s) — %d KBs, %d agents, %d skills",
		srv.Name, srv.BaseURL, len(caps.KnowledgeBases), len(caps.Agents), len(caps.Skills))
	return nil
}

// ProvisionUser auto-registers the client's local user onto the server and
// obtains a server JWT for that user. Steps:
//  1. discover the API token's tenant via GET /auth/me,
//  2. provision the user via POST /tenants/:id/client-users (API token),
//  3. log in as the user via POST /auth/login to obtain a JWT + refresh token,
//  4. persist the linked identity (encrypted) and update the live connection.
func (c *Connector) ProvisionUser(ctx context.Context, serverID, localEmail, localUsername string) error {
	srv, err := c.store.GetServer(ctx, serverID)
	if err != nil {
		return fmt.Errorf("enterprise: server not found: %w", err)
	}
	client := &http.Client{Timeout: defaultRequestTimeout}
	base := strings.TrimRight(srv.BaseURL, "/")

	tenantID, err := discoverTenantID(ctx, client, base, srv.APIToken)
	if err != nil {
		return fmt.Errorf("enterprise: discover tenant failed: %w", err)
	}

	password, err := generateClientPassword()
	if err != nil {
		return fmt.Errorf("enterprise: generate password failed: %w", err)
	}

	provisioned, err := provisionClientUser(ctx, client, base, srv.APIToken, tenantID, localEmail, localUsername, password)
	if err != nil {
		return fmt.Errorf("enterprise: provision user failed: %w", err)
	}

	jwt, refresh, err := serverLogin(ctx, client, base, localEmail, password)
	if err != nil {
		return fmt.Errorf("enterprise: server login failed: %w", err)
	}

	linkedTenantID := provisioned.TenantID
	if linkedTenantID == "" {
		linkedTenantID = tenantID
	}

	if err := c.store.SaveLinkedIdentity(ctx, serverID, provisioned.UserID, provisioned.Email, linkedTenantID, password, jwt, refresh); err != nil {
		return fmt.Errorf("enterprise: save linked identity failed: %w", err)
	}

	if conn := c.GetConnection(serverID); conn != nil {
		conn.Config.LinkedUserID = provisioned.UserID
		conn.Config.LinkedEmail = provisioned.Email
		conn.Config.LinkedTenantID = linkedTenantID
		conn.Config.LinkedPassword = password
		conn.Config.ServerJWT = jwt
		conn.Config.ServerRefreshToken = refresh
	}

	logger.Infof(ctx, "enterprise: provisioned user %q into home tenant %s on %q (api token tenant %s)",
		provisioned.Email, linkedTenantID, srv.Name, tenantID)
	return nil
}

// provisionedUser is the subset of the provisioning response the client needs.
type provisionedUser struct {
	UserID   string
	Email    string
	TenantID string
}

// discoverTenantID calls GET /auth/me with the API token and returns the active
// tenant id (the tenant the API token is bound to).
func discoverTenantID(ctx context.Context, client *http.Client, base, apiToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/auth/me", nil)
	if err != nil {
		return "", err
	}
	if apiToken != "" {
		req.Header.Set("X-API-Key", apiToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Data struct {
			Tenant struct {
				ID json.Number `json:"id"`
			} `json:"tenant"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	id := parsed.Data.Tenant.ID.String()
	if id == "" || id == "0" {
		return "", fmt.Errorf("no tenant id in /auth/me response")
	}
	return id, nil
}

// provisionClientUser calls POST /tenants/:id/client-users with the API token.
func provisionClientUser(ctx context.Context, client *http.Client, base, apiToken, tenantID, email, username, password string) (*provisionedUser, error) {
	payload, _ := json.Marshal(map[string]string{
		"email":    email,
		"username": username,
		"password": password,
		"role":     "contributor",
	})
	url := fmt.Sprintf("%s/api/v1/tenants/%s/client-users", base, tenantID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	if apiToken != "" {
		req.Header.Set("X-API-Key", apiToken)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Data struct {
			UserID   string      `json:"user_id"`
			Email    string      `json:"email"`
			TenantID json.Number `json:"tenant_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return &provisionedUser{
		UserID:   parsed.Data.UserID,
		Email:    parsed.Data.Email,
		TenantID: parsed.Data.TenantID.String(),
	}, nil
}

// serverLogin calls POST /auth/login and returns the access + refresh tokens.
func serverLogin(ctx context.Context, client *http.Client, base, email, password string) (string, string, error) {
	payload, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/auth/login", bytes.NewReader(payload))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", "", err
	}
	if parsed.Token == "" {
		return "", "", fmt.Errorf("login returned no token")
	}
	return parsed.Token, parsed.RefreshToken, nil
}

// generateClientPassword produces a random URL-safe password owned by the client.
func generateClientPassword() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Disconnect tears down the connection to the given server.
func (c *Connector) Disconnect(serverID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if conn, ok := c.connections[serverID]; ok {
		conn.Status = StatusDisconnected
		delete(c.connections, serverID)
	}
	c.emitHealth(serverID, StatusDisconnected, "")
}

// GetConnection returns the live connection for a server, or nil if not connected.
func (c *Connector) GetConnection(serverID string) *ServerConnection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connections[serverID]
}

// ListConnections returns a snapshot of all active connections.
func (c *Connector) ListConnections() []*ServerConnection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]*ServerConnection, 0, len(c.connections))
	for _, conn := range c.connections {
		result = append(result, conn)
	}
	return result
}

// TestConnection verifies that a server is reachable without persisting state.
func (c *Connector) TestConnection(ctx context.Context, baseURL, apiToken string) error {
	client := &http.Client{Timeout: defaultConnectTimeout}
	url := strings.TrimRight(baseURL, "/") + "/api/v1/system/info"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if apiToken != "" {
		req.Header.Set("X-API-Key", apiToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// --- internal ---

func (c *Connector) discover(ctx context.Context, conn *ServerConnection) (*ServerCapabilities, error) {
	base := strings.TrimRight(conn.Config.BaseURL, "/")

	// Prefer the linked user JWT so the server resolves the client user's
	// own home tenant; fall back to the tenant-scoped API token before the
	// user is provisioned (initial connect). Shared resources are merged
	// separately via RefreshSharedResources.
	jwt := conn.Config.ServerJWT
	apiToken := conn.Config.APIToken

	// Knowledge bases: own home-tenant resources.
	kbs, kbErr := fetchOwnKnowledgeBases(ctx, conn.client, base, jwt, apiToken)
	if kbErr != nil {
		return nil, fmt.Errorf("fetch knowledge bases: %w", kbErr)
	}

	// Agents: own home-tenant agents.
	agents, agentErr := fetchOwnAgents(ctx, conn.client, base, jwt, apiToken)
	if agentErr != nil {
		// Non-fatal: some servers may not expose agents.
		logger.Warnf(ctx, "enterprise: fetch agents from %q: %v", conn.Config.Name, agentErr)
	}

	// Skills: server-wide preloaded skills (not tenant-scoped).
	skills, skillErr := fetchOwnSkills(ctx, conn.client, base, jwt, apiToken)
	if skillErr != nil {
		logger.Warnf(ctx, "enterprise: fetch skills from %q: %v", conn.Config.Name, skillErr)
	}

	return &ServerCapabilities{
		KnowledgeBases: kbs,
		Agents:         agents,
		Skills:         skills,
	}, nil
}

// Rediscover re-fetches the linked user's own home-tenant resources (KBs,
// agents, skills) using the linked JWT and replaces the "own" (non-shared)
// portion of the connection's capabilities. Shared entries (Shared=true) are
// preserved so a re-discover does not drop org-shared resources. Used after
// ProvisionUser so the client sees the freshly-linked user's own workspace
// instead of the admin tenant's resources left over from the initial connect.
func (c *Connector) Rediscover(ctx context.Context, serverID string) error {
	conn := c.GetConnection(serverID)
	if conn == nil {
		return fmt.Errorf("enterprise: not connected")
	}
	own, err := c.discover(ctx, conn)
	if err != nil {
		return fmt.Errorf("enterprise: rediscover failed: %w", err)
	}
	if conn.Capabilities == nil {
		conn.Capabilities = &ServerCapabilities{}
	}
	conn.Capabilities.KnowledgeBases = append(dropSharedKBs(own.KnowledgeBases), keepSharedKBs(conn.Capabilities.KnowledgeBases)...)
	conn.Capabilities.Agents = append(dropSharedAgents(own.Agents), keepSharedAgents(conn.Capabilities.Agents)...)
	conn.Capabilities.Skills = own.Skills
	c.cacheCapabilities(ctx, serverID, conn.Capabilities)
	logger.Infof(ctx, "enterprise: re-discovered own resources from %q — %d KBs, %d agents, %d skills",
		conn.Config.Name, len(own.KnowledgeBases), len(own.Agents), len(own.Skills))
	return nil
}

// fetchJSONWithJWT performs a GET request authenticated with a Bearer JWT and
// decodes the JSON response into out.
func fetchJSONWithJWT(ctx context.Context, client *http.Client, url, jwt string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// fetchSharedKnowledgeBases calls GET /shared-knowledge-bases with the linked JWT.
func fetchSharedKnowledgeBases(ctx context.Context, client *http.Client, base, jwt string) ([]RemoteKnowledgeBase, error) {
	var parsed struct {
		Data []struct {
			KnowledgeBase struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"knowledge_base"`
			Permission string `json:"permission"`
			OrgName    string `json:"org_name"`
		} `json:"data"`
	}
	if err := fetchJSONWithJWT(ctx, client, base+"/api/v1/shared-knowledge-bases", jwt, &parsed); err != nil {
		return nil, err
	}
	out := make([]RemoteKnowledgeBase, 0, len(parsed.Data))
	for _, row := range parsed.Data {
		out = append(out, RemoteKnowledgeBase{
			ID:          row.KnowledgeBase.ID,
			Name:        row.KnowledgeBase.Name,
			Description: row.KnowledgeBase.Description,
			Shared:      true,
			Permission:  row.Permission,
			OrgName:     row.OrgName,
		})
	}
	return out, nil
}

// fetchSharedAgents calls GET /shared-agents with the linked JWT, skipping
// agents the user has disabled.
func fetchSharedAgents(ctx context.Context, client *http.Client, base, jwt string) ([]RemoteAgent, error) {
	var parsed struct {
		Data []struct {
			Agent struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
				Avatar      string `json:"avatar"`
			} `json:"agent"`
			Permission   string `json:"permission"`
			OrgName      string `json:"org_name"`
			DisabledByMe bool   `json:"disabled_by_me"`
		} `json:"data"`
	}
	if err := fetchJSONWithJWT(ctx, client, base+"/api/v1/shared-agents", jwt, &parsed); err != nil {
		return nil, err
	}
	out := make([]RemoteAgent, 0, len(parsed.Data))
	for _, row := range parsed.Data {
		if row.DisabledByMe {
			continue
		}
		out = append(out, RemoteAgent{
			ID:          row.Agent.ID,
			Name:        row.Agent.Name,
			Description: row.Agent.Description,
			AvatarURL:   row.Agent.Avatar,
			Shared:      true,
			Permission:  row.Permission,
			OrgName:     row.OrgName,
		})
	}
	return out, nil
}

func dropSharedKBs(kbs []RemoteKnowledgeBase) []RemoteKnowledgeBase {
	out := make([]RemoteKnowledgeBase, 0, len(kbs))
	for _, kb := range kbs {
		if !kb.Shared {
			out = append(out, kb)
		}
	}
	return out
}

func dropSharedAgents(agents []RemoteAgent) []RemoteAgent {
	out := make([]RemoteAgent, 0, len(agents))
	for _, a := range agents {
		if !a.Shared {
			out = append(out, a)
		}
	}
	return out
}

// keepSharedKBs returns only the org-shared knowledge bases (Shared=true),
// the inverse of dropSharedKBs. Rediscover preserves these across re-discovery.
func keepSharedKBs(kbs []RemoteKnowledgeBase) []RemoteKnowledgeBase {
	out := make([]RemoteKnowledgeBase, 0, len(kbs))
	for _, kb := range kbs {
		if kb.Shared {
			out = append(out, kb)
		}
	}
	return out
}

// keepSharedAgents returns only the org-shared agents (Shared=true).
func keepSharedAgents(agents []RemoteAgent) []RemoteAgent {
	out := make([]RemoteAgent, 0, len(agents))
	for _, a := range agents {
		if a.Shared {
			out = append(out, a)
		}
	}
	return out
}

// fetchJSONWithAuth performs a GET request authenticated with the linked JWT
// when available, falling back to the tenant-scoped API token, and decodes the
// JSON response (wrapped as {"data": ...}) into out.
func fetchJSONWithAuth(ctx context.Context, client *http.Client, url, jwt, apiToken string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	} else if apiToken != "" {
		req.Header.Set("X-API-Key", apiToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// fetchOwnKnowledgeBases lists the linked user own home-tenant knowledge bases
// via GET /knowledge-bases (the server resolves the caller tenant from the JWT).
// Shared=false distinguishes these from org-shared entries.
func fetchOwnKnowledgeBases(ctx context.Context, client *http.Client, base, jwt, apiToken string) ([]RemoteKnowledgeBase, error) {
	var parsed struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"data"`
	}
	if err := fetchJSONWithAuth(ctx, client, base+"/api/v1/knowledge-bases", jwt, apiToken, &parsed); err != nil {
		return nil, err
	}
	out := make([]RemoteKnowledgeBase, 0, len(parsed.Data))
	for _, kb := range parsed.Data {
		out = append(out, RemoteKnowledgeBase{ID: kb.ID, Name: kb.Name, Description: kb.Description})
	}
	return out, nil
}

// fetchOwnAgents lists the linked user own home-tenant agents via GET /agents.
func fetchOwnAgents(ctx context.Context, client *http.Client, base, jwt, apiToken string) ([]RemoteAgent, error) {
	var parsed struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Avatar      string `json:"avatar"`
		} `json:"data"`
	}
	if err := fetchJSONWithAuth(ctx, client, base+"/api/v1/agents", jwt, apiToken, &parsed); err != nil {
		return nil, err
	}
	out := make([]RemoteAgent, 0, len(parsed.Data))
	for _, a := range parsed.Data {
		out = append(out, RemoteAgent{ID: a.ID, Name: a.Name, Description: a.Description, AvatarURL: a.Avatar})
	}
	return out, nil
}

// fetchOwnSkills lists the server preloaded skills via GET /skills.
func fetchOwnSkills(ctx context.Context, client *http.Client, base, jwt, apiToken string) ([]RemoteSkill, error) {
	var parsed struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"data"`
	}
	if err := fetchJSONWithAuth(ctx, client, base+"/api/v1/skills", jwt, apiToken, &parsed); err != nil {
		return nil, err
	}
	out := make([]RemoteSkill, 0, len(parsed.Data))
	for _, sk := range parsed.Data {
		out = append(out, RemoteSkill{ID: sk.ID, Name: sk.Name, Description: sk.Description})
	}
	return out, nil
}

// RefreshSharedResources fetches the resources shared with the linked user (via
// organizations) using the linked JWT and merges them into the connection's
// capabilities, replacing any previously-fetched shared entries. Requires a
// linked identity (the user must have been provisioned).
func (c *Connector) RefreshSharedResources(ctx context.Context, serverID string) error {
	conn := c.GetConnection(serverID)
	if conn == nil {
		return fmt.Errorf("enterprise: not connected")
	}
	if conn.Config.ServerJWT == "" {
		return fmt.Errorf("enterprise: no linked identity; provision first")
	}
	base := strings.TrimRight(conn.Config.BaseURL, "/")

	sharedKBs, kbErr := fetchSharedKnowledgeBases(ctx, conn.client, base, conn.Config.ServerJWT)
	if kbErr != nil {
		logger.Warnf(ctx, "enterprise: fetch shared KBs from %q: %v", conn.Config.Name, kbErr)
	}
	sharedAgents, agentErr := fetchSharedAgents(ctx, conn.client, base, conn.Config.ServerJWT)
	if agentErr != nil {
		logger.Warnf(ctx, "enterprise: fetch shared agents from %q: %v", conn.Config.Name, agentErr)
	}

	if conn.Capabilities == nil {
		conn.Capabilities = &ServerCapabilities{}
	}
	conn.Capabilities.KnowledgeBases = append(dropSharedKBs(conn.Capabilities.KnowledgeBases), sharedKBs...)
	conn.Capabilities.Agents = append(dropSharedAgents(conn.Capabilities.Agents), sharedAgents...)

	c.cacheCapabilities(ctx, serverID, conn.Capabilities)
	logger.Infof(ctx, "enterprise: refreshed shared resources from %q — %d KBs, %d agents",
		conn.Config.Name, len(sharedKBs), len(sharedAgents))
	return nil
}

func (c *Connector) cacheCapabilities(ctx context.Context, serverID string, caps *ServerCapabilities) {
	var entries []ResourceCacheEntry
	now := time.Now()
	for _, kb := range caps.KnowledgeBases {
		meta, _ := json.Marshal(kb)
		entries = append(entries, ResourceCacheEntry{
			ID:           fmt.Sprintf("%s-kb-%s", serverID, kb.ID),
			ServerID:     serverID,
			ResourceType: "knowledge_base",
			ResourceID:   kb.ID,
			Metadata:     string(meta),
			CachedAt:     now,
		})
	}
	for _, agent := range caps.Agents {
		meta, _ := json.Marshal(agent)
		entries = append(entries, ResourceCacheEntry{
			ID:           fmt.Sprintf("%s-agent-%s", serverID, agent.ID),
			ServerID:     serverID,
			ResourceType: "agent",
			ResourceID:   agent.ID,
			Metadata:     string(meta),
			CachedAt:     now,
		})
	}
	for _, skill := range caps.Skills {
		meta, _ := json.Marshal(skill)
		entries = append(entries, ResourceCacheEntry{
			ID:           fmt.Sprintf("%s-skill-%s", serverID, skill.ID),
			ServerID:     serverID,
			ResourceType: "skill",
			ResourceID:   skill.ID,
			Metadata:     string(meta),
			CachedAt:     now,
		})
	}
	if err := c.store.CacheResources(ctx, serverID, entries); err != nil {
		logger.Warnf(ctx, "enterprise: failed to cache resources for %s: %v", serverID, err)
	}
}

func (c *Connector) healthLoop() {
	ticker := time.NewTicker(defaultHealthInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.mu.RLock()
			ids := make([]string, 0, len(c.connections))
			for id := range c.connections {
				ids = append(ids, id)
			}
			c.mu.RUnlock()

			now := time.Now()
			for _, id := range ids {
				conn := c.GetConnection(id)
				if conn == nil {
					continue
				}
				// Exponential backoff: skip servers whose retry window hasn't opened.
				if conn.Status == StatusError && now.Before(conn.nextRetryAt) {
					continue
				}
				ctx, cancel := context.WithTimeout(context.Background(), defaultConnectTimeout)
				err := c.TestConnection(ctx, conn.Config.BaseURL, conn.Config.APIToken)
				cancel()
				if err != nil {
					wasConnected := conn.Status == StatusConnected
					conn.Status = StatusError
					conn.LastError = err.Error()
					// Grow backoff on repeated failures, reset when we were healthy.
					if wasConnected || conn.currentBackoff == 0 {
						conn.currentBackoff = initialBackoff
					} else {
						conn.currentBackoff = time.Duration(float64(conn.currentBackoff) * backoffFactor)
						if conn.currentBackoff > maxBackoff {
							conn.currentBackoff = maxBackoff
						}
					}
					conn.nextRetryAt = now.Add(conn.currentBackoff)
					c.emitHealth(id, StatusError, err.Error())
				} else {
					if conn.Status != StatusConnected {
						// Recovered: refresh capabilities so the resource list is current.
						if caps, derr := c.discover(ctx, conn); derr == nil {
							conn.Capabilities = caps
							c.cacheCapabilities(context.Background(), id, caps)
						}
					}
					conn.Status = StatusConnected
					conn.LastError = ""
					conn.LastSyncedAt = now
					conn.currentBackoff = 0
					conn.nextRetryAt = time.Time{}
					_ = c.store.TouchServer(context.Background(), id)
					c.emitHealth(id, StatusConnected, "")
				}
			}
		}
	}
}

func (c *Connector) emitHealth(serverID string, status ConnectionStatus, errMsg string) {
	select {
	case c.healthCh <- HealthEvent{ServerID: serverID, Status: status, Error: errMsg, Time: time.Now()}:
	default:
	}
}

// fetchRemoteList performs a GET request and decodes a JSON array response.
func fetchRemoteList[T any](ctx context.Context, client *http.Client, url, apiToken string) ([]T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if apiToken != "" {
		req.Header.Set("X-API-Key", apiToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	var items []T
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return items, nil
}
