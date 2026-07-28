package middleware

import (
	"net/http"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/gin-gonic/gin"
)

// RequireCapability denies the request unless the authenticated user has the
// given per-user capability flag. It runs AFTER tenant RBAC so role errors
// stay unchanged when the role itself is insufficient.
//
// Prefer the User already placed on context by Auth middleware. When absent
// (tests / unusual paths), deny rather than allow.
func RequireCapability(id string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := c.Request.Context().Value(types.UserContextKey).(*types.User)
		if user == nil {
			// Fallback: gin Keys set by Auth (string key form).
			if v, ok := c.Get(types.UserContextKey.String()); ok {
				user, _ = v.(*types.User)
			}
		}
		if !types.UserHasCapability(user, id) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    "capability_denied",
				"message": "Current account lacks this permission; contact an administrator",
			})
			return
		}
		c.Next()
	}
}
