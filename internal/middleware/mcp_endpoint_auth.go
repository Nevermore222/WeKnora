package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Tencent/Xelora/internal/application/service"
	"github.com/Tencent/Xelora/internal/logger"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// MCPEndpointAuth authenticates the public built-in MCP endpoint with its
// bearer token and installs the same tenant/user context consumed by existing
// Xelora services. Query-string tokens are deliberately not accepted.
func MCPEndpointAuth(svc interfaces.MCPEndpointService, tenantSvc interfaces.TenantService) gin.HandlerFunc {
	return func(c *gin.Context) {
		endpointID := strings.TrimSpace(c.Param("endpoint_id"))
		token := extractMCPEndpointToken(c)
		if endpointID == "" || token == "" {
			c.Header("WWW-Authenticate", `Bearer realm="xelora-mcp"`)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing bearer token"})
			c.Abort()
			return
		}
		ep, err := svc.Authenticate(c.Request.Context(), endpointID, token)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrMCPEndpointDisabled):
				c.JSON(http.StatusForbidden, gin.H{"error": "mcp endpoint disabled"})
			case errors.Is(err, service.ErrMCPEndpointTokenInvalid), errors.Is(err, service.ErrMCPEndpointNotFound):
				c.Header("WWW-Authenticate", `Bearer realm="xelora-mcp"`)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: invalid token"})
			default:
				logger.Warnf(c.Request.Context(), "[mcp-endpoint] auth lookup failed: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication unavailable"})
			}
			c.Abort()
			return
		}
		if tenantSvc == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "workspace service unavailable"})
			c.Abort()
			return
		}
		tenant, err := tenantSvc.GetTenantByID(c.Request.Context(), ep.TenantID)
		if err != nil || tenant == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "workspace unavailable"})
			c.Abort()
			return
		}
		if strings.EqualFold(tenant.Status, "inactive") || strings.EqualFold(tenant.Status, "disabled") {
			c.JSON(http.StatusForbidden, gin.H{"error": "workspace inactive"})
			c.Abort()
			return
		}

		user := &types.User{
			ID: fmt.Sprintf("mcp-%s", ep.ID), Username: fmt.Sprintf("mcp-%s", ep.ID),
			Email: fmt.Sprintf("mcp-%s@mcp.local", ep.ID), TenantID: ep.TenantID, IsActive: true,
		}
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, types.TenantIDContextKey, ep.TenantID)
		ctx = context.WithValue(ctx, types.CallerTenantIDContextKey, ep.TenantID)
		ctx = context.WithValue(ctx, types.TenantInfoContextKey, tenant)
		ctx = context.WithValue(ctx, types.UserContextKey, user)
		ctx = context.WithValue(ctx, types.UserIDContextKey, user.ID)
		ctx = context.WithValue(ctx, types.TenantRoleContextKey, types.TenantRoleViewer)
		ctx = context.WithValue(ctx, types.SystemAdminContextKey, false)
		ctx = context.WithValue(ctx, types.MCPEndpointContextKey, ep)
		c.Set(types.TenantIDContextKey.String(), ep.TenantID)
		c.Set(types.CallerTenantIDContextKey.String(), ep.TenantID)
		c.Set(types.TenantInfoContextKey.String(), tenant)
		c.Set(types.UserContextKey.String(), user)
		c.Set(types.UserIDContextKey.String(), user.ID)
		c.Set(types.TenantRoleContextKey.String(), types.TenantRoleViewer)
		c.Set(types.SystemAdminContextKey.String(), false)
		c.Set(types.MCPEndpointContextKey.String(), ep)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func extractMCPEndpointToken(c *gin.Context) string {
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func MCPEndpointFromContext(ctx context.Context) (*types.MCPEndpoint, bool) {
	ep, ok := ctx.Value(types.MCPEndpointContextKey).(*types.MCPEndpoint)
	return ep, ok && ep != nil
}
