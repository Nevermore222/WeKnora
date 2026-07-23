package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/Tencent/Xelora/internal/enterprise"
	"github.com/Tencent/Xelora/internal/logger"
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// EnterpriseHandler exposes REST endpoints for managing enterprise server
// connections and proxying resource requests to remote Xelora servers.
type EnterpriseHandler struct {
	store       *enterprise.Store
	connector   *enterprise.Connector
	proxy       *enterprise.Proxy
	userService interfaces.UserService
}

// NewEnterpriseHandler creates the handler with its dependencies.
func NewEnterpriseHandler(store *enterprise.Store, connector *enterprise.Connector, proxy *enterprise.Proxy, userService interfaces.UserService) *EnterpriseHandler {
	return &EnterpriseHandler{store: store, connector: connector, proxy: proxy, userService: userService}
}

// --- Server CRUD ---

type createServerRequest struct {
	Name        string `json:"name" binding:"required"`
	BaseURL     string `json:"base_url" binding:"required"`
	APIToken    string `json:"api_token"`
	AutoConnect *bool  `json:"auto_connect"`
}

type updateServerRequest struct {
	Name        *string `json:"name"`
	BaseURL     *string `json:"base_url"`
	APIToken    *string `json:"api_token"`
	AutoConnect *bool   `json:"auto_connect"`
}

// ListServers returns all configured enterprise servers.
func (h *EnterpriseHandler) ListServers(c *gin.Context) {
	servers, err := h.store.ListServers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Enrich with live connection status.
	type serverResponse struct {
		enterprise.ServerConfig
		Status    string `json:"status"`
		LastError string `json:"last_error,omitempty"`
	}
	result := make([]serverResponse, 0, len(servers))
	for _, srv := range servers {
		resp := serverResponse{ServerConfig: srv, Status: string(enterprise.StatusDisconnected)}
		if conn := h.connector.GetConnection(srv.ID); conn != nil {
			resp.Status = string(conn.Status)
			resp.LastError = conn.LastError
		}
		result = append(result, resp)
	}
	c.JSON(http.StatusOK, result)
}

// CreateServer adds a new enterprise server configuration.
func (h *EnterpriseHandler) CreateServer(c *gin.Context) {
	var req createServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and base_url are required"})
		return
	}
	autoConnect := true
	if req.AutoConnect != nil {
		autoConnect = *req.AutoConnect
	}
	srv, err := h.store.CreateServer(c.Request.Context(), req.Name, strings.TrimRight(req.BaseURL, "/"), req.APIToken, autoConnect)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, srv)
}

// UpdateServer modifies an existing server configuration.
func (h *EnterpriseHandler) UpdateServer(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req updateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.BaseURL != nil {
		trimmed := strings.TrimRight(*req.BaseURL, "/")
		req.BaseURL = &trimmed
	}
	srv, err := h.store.UpdateServer(c.Request.Context(), id, req.Name, req.BaseURL, req.APIToken, req.AutoConnect)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}
	c.JSON(http.StatusOK, srv)
}

// DeleteServer removes a server and disconnects it.
func (h *EnterpriseHandler) DeleteServer(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	h.connector.Disconnect(id)
	if err := h.store.DeleteServer(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// TestServer verifies connectivity without persisting state.
func (h *EnterpriseHandler) TestServer(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	srv, err := h.store.GetServer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}
	if err := h.connector.TestConnection(c.Request.Context(), srv.BaseURL, srv.APIToken); err != nil {
		c.JSON(http.StatusOK, gin.H{"reachable": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reachable": true})
}

// DiscoverServers browses the LAN for Xelora servers via mDNS.
func (h *EnterpriseHandler) DiscoverServers(c *gin.Context) {
	servers, err := enterprise.DiscoverServers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"servers": []enterprise.DiscoveredServer{}, "note": err.Error()})
		return
	}
	if servers == nil {
		servers = []enterprise.DiscoveredServer{}
	}
	c.JSON(http.StatusOK, gin.H{"servers": servers})
}

// ConnectServer initiates a connection to the server, then auto-provisions the
// current local user onto the server so the admin can share resources with them.
// Provisioning is best-effort: a failure logs a warning but does not fail the
// connect (the server may already have the user, or provisioning can be retried).
func (h *EnterpriseHandler) ConnectServer(c *gin.Context) {
	ctx := c.Request.Context()
	id := strings.TrimSpace(c.Param("id"))
	if err := h.connector.Connect(ctx, id); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	linkedEmail := ""
	if user, err := h.userService.GetCurrentUser(ctx); err == nil && user != nil {
		if perr := h.connector.ProvisionUser(ctx, id, user.Email, user.Username); perr != nil {
			logger.Warnf(ctx, "enterprise: auto-provision after connect failed for server %s: %v", id, perr)
		} else {
			if conn := h.connector.GetConnection(id); conn != nil {
				linkedEmail = conn.Config.LinkedEmail
			}
			// The initial Connect used the API token (no JWT yet), so its
			// discovered "own" resources were the admin tenant's. Re-discover
			// with the freshly-linked JWT so capabilities reflect the client
			// user's own home tenant, then pull org-shared resources.
			if derr := h.connector.Rediscover(ctx, id); derr != nil {
				logger.Warnf(ctx, "enterprise: re-discover own resources failed for server %s: %v", id, derr)
			}
			if serr := h.connector.RefreshSharedResources(ctx, id); serr != nil {
				logger.Warnf(ctx, "enterprise: refresh shared resources failed for server %s: %v", id, serr)
			}
		}
	} else {
		logger.Warnf(ctx, "enterprise: cannot resolve local user for provisioning: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"status": "connected", "linked_email": linkedEmail})
}

// DisconnectServer tears down the connection.
func (h *EnterpriseHandler) DisconnectServer(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	h.connector.Disconnect(id)
	c.JSON(http.StatusOK, gin.H{"status": "disconnected"})
}

// ServerStatus returns the live connection status for a server.
func (h *EnterpriseHandler) ServerStatus(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	conn := h.connector.GetConnection(id)
	if conn == nil {
		c.JSON(http.StatusOK, gin.H{"status": "disconnected"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":         string(conn.Status),
		"last_error":     conn.LastError,
		"last_synced_at": conn.LastSyncedAt,
		"linked_email":   conn.Config.LinkedEmail,
	})
}

// --- Aggregated Resources ---

// ListResources returns enterprise resources from all connected servers.
func (h *EnterpriseHandler) ListResources(c *gin.Context) {
	resourceType := c.Query("type")
	connections := h.connector.ListConnections()
	var resources []enterprise.AggregatedResource

	for _, conn := range connections {
		if conn.Capabilities == nil {
			continue
		}
		serverName := conn.Config.Name
		serverID := conn.Config.ID
		available := conn.Status == enterprise.StatusConnected

		if resourceType == "" || resourceType == "knowledge_base" {
			for _, kb := range conn.Capabilities.KnowledgeBases {
				resources = append(resources, enterprise.AggregatedResource{
					ID:          kb.ID,
					Type:        "knowledge_base",
					Name:        kb.Name,
					Description: kb.Description,
					Origin:      "enterprise",
					ServerID:    serverID,
					ServerName:  serverName,
					Available:   available,
					Shared:      kb.Shared,
					Permission:  kb.Permission,
					OrgName:     kb.OrgName,
				})
			}
		}
		if resourceType == "" || resourceType == "agent" {
			for _, agent := range conn.Capabilities.Agents {
				resources = append(resources, enterprise.AggregatedResource{
					ID:          agent.ID,
					Type:        "agent",
					Name:        agent.Name,
					Description: agent.Description,
					Origin:      "enterprise",
					ServerID:    serverID,
					ServerName:  serverName,
					Available:   available,
					Shared:      agent.Shared,
					Permission:  agent.Permission,
					OrgName:     agent.OrgName,
				})
			}
		}
		if resourceType == "" || resourceType == "skill" {
			for _, skill := range conn.Capabilities.Skills {
				resources = append(resources, enterprise.AggregatedResource{
					ID:          skill.ID,
					Type:        "skill",
					Name:        skill.Name,
					Description: skill.Description,
					Origin:      "enterprise",
					ServerID:    serverID,
					ServerName:  serverName,
					Available:   available,
				})
			}
		}
	}

	if resources == nil {
		resources = []enterprise.AggregatedResource{}
	}
	c.JSON(http.StatusOK, gin.H{"resources": resources})
}

// --- Proxy Endpoints ---

// ProxyChat forwards a chat request to an enterprise agent.
func (h *EnterpriseHandler) ProxyChat(c *gin.Context) {
	serverID := c.GetHeader("X-Enterprise-Server-ID")
	conn := h.connector.GetConnection(serverID)
	if conn == nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "enterprise server not connected"})
		return
	}

	sessionID := strings.TrimSpace(c.Param("session_id"))
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	body, _ := io.ReadAll(c.Request.Body)
	resp, err := h.proxy.ForwardSSE(c.Request.Context(), conn, "/api/v1/agent-chat/"+sessionID, body, c.Request.Header)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	// Stream the SSE response back to the client.
	c.Status(resp.StatusCode)
	for key, vals := range resp.Header {
		for _, v := range vals {
			c.Header(key, v)
		}
	}
	_, _ = io.Copy(c.Writer, resp.Body)
}

// ProxyCreateSession creates a session on the enterprise server (so enterprise
// agent chats have a server-side session for multi-turn context) and returns the
// server's response, including the new session id.
func (h *EnterpriseHandler) ProxyCreateSession(c *gin.Context) {
	serverID := c.GetHeader("X-Enterprise-Server-ID")
	conn := h.connector.GetConnection(serverID)
	if conn == nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "enterprise server not connected"})
		return
	}

	body, _ := io.ReadAll(c.Request.Body)
	resp, err := h.proxy.ForwardJSON(c.Request.Context(), conn, http.MethodPost, "/api/v1/sessions", body, c.Request.Header)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
	c.Header("Content-Type", "application/json")
	_, _ = io.Copy(c.Writer, resp.Body)
}

// ProxyRetrieval forwards a knowledge base retrieval request.
func (h *EnterpriseHandler) ProxyRetrieval(c *gin.Context) {
	serverID := c.GetHeader("X-Enterprise-Server-ID")
	conn := h.connector.GetConnection(serverID)
	if conn == nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "enterprise server not connected"})
		return
	}

	body, _ := io.ReadAll(c.Request.Body)
	resp, err := h.proxy.ForwardJSON(c.Request.Context(), conn, http.MethodPost, "/api/v1/retrieval", body, c.Request.Header)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
	c.Header("Content-Type", "application/json")
	_, _ = io.Copy(c.Writer, resp.Body)
}

// ProxySkillExecute forwards a skill execution request to the enterprise server.
func (h *EnterpriseHandler) ProxySkillExecute(c *gin.Context) {
	serverID := c.GetHeader("X-Enterprise-Server-ID")
	conn := h.connector.GetConnection(serverID)
	if conn == nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "enterprise server not connected"})
		return
	}

	body, _ := io.ReadAll(c.Request.Body)
	resp, err := h.proxy.ForwardJSON(c.Request.Context(), conn, http.MethodPost, "/api/v1/skill/execute", body, c.Request.Header)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
	c.Header("Content-Type", "application/json")
	_, _ = io.Copy(c.Writer, resp.Body)
}

// --- KB & Agent proxy CRUD ---
//
// These forward resource-management requests to the enterprise server using
// the linked user JWT (applyAuth prefers it; ForwardJSON auto-refreshs on
// 401). The server enforces ownership/role gates against the caller
// (user+tenant resolved from the JWT), so the client needs no local
// authorization logic: a linked user can create/list/edit/delete resources
// only within their own home tenant, exactly as a normal server user would.

// proxyKB forwards a knowledge-base CRUD request (JSON body) to the server's
// /knowledge-bases routes. The :id path param (when present) is forwarded.
func (h *EnterpriseHandler) ProxyKB(c *gin.Context) {
	conn := h.enterpriseConn(c)
	if conn == nil {
		return
	}
	path := "/api/v1/knowledge-bases"
	if id := c.Param("id"); id != "" {
		path += "/" + id
	}
	body, _ := io.ReadAll(c.Request.Body)
	resp, err := h.proxy.ForwardJSON(c.Request.Context(), conn, c.Request.Method, path, body, c.Request.Header)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	c.Status(resp.StatusCode)
	c.Header("Content-Type", "application/json")
	_, _ = io.Copy(c.Writer, resp.Body)
}

// proxyKBKnowledge forwards knowledge (document) sub-resource requests for a
// KB: list documents (GET) or upload a document (multipart POST, streamed via
// ForwardRequest so the original Content-Type/boundary is preserved).
func (h *EnterpriseHandler) ProxyKBKnowledge(c *gin.Context) {
	conn := h.enterpriseConn(c)
	if conn == nil {
		return
	}
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "knowledge base id is required"})
		return
	}
	path := "/api/v1/knowledge-bases/" + id + "/knowledge"
	resp, err := h.proxy.ForwardRequest(c.Request.Context(), conn, c.Request.Method, path, c.Request.Body, c.Request.Header)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	c.Status(resp.StatusCode)
	for key, vals := range resp.Header {
		for _, v := range vals {
			if key == "Content-Length" || key == "Transfer-Encoding" {
				continue
			}
			c.Header(key, v)
		}
	}
	_, _ = io.Copy(c.Writer, resp.Body)
}

// proxyAgent forwards an agent CRUD request (JSON body) to the server's
// /agents routes. The :id path param (when present) is forwarded.
func (h *EnterpriseHandler) ProxyAgent(c *gin.Context) {
	conn := h.enterpriseConn(c)
	if conn == nil {
		return
	}
	path := "/api/v1/agents"
	if id := c.Param("id"); id != "" {
		path += "/" + id
	}
	body, _ := io.ReadAll(c.Request.Body)
	resp, err := h.proxy.ForwardJSON(c.Request.Context(), conn, c.Request.Method, path, body, c.Request.Header)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	c.Status(resp.StatusCode)
	c.Header("Content-Type", "application/json")
	_, _ = io.Copy(c.Writer, resp.Body)
}

// enterpriseConn resolves the target connection from the X-Enterprise-Server-ID
// header, writing a 502 error when it is missing or disconnected.
func (h *EnterpriseHandler) enterpriseConn(c *gin.Context) *enterprise.ServerConnection {
	serverID := c.GetHeader("X-Enterprise-Server-ID")
	conn := h.connector.GetConnection(serverID)
	if conn == nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "enterprise server not connected"})
	}
	return conn
}
