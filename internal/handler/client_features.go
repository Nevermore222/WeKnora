package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ClientFeaturesResponse struct {
	Features map[string]bool `json:"features"`
}

// GetClientFeatures exposes authenticated, server-controlled client feature flags.
// Unknown or malformed settings stay disabled so a client cannot enable a
// hidden workflow because the control-plane value is unavailable.
func (h *SystemHandler) GetClientFeatures(c *gin.Context) {
	enabled := false
	if h != nil && h.systemSettingSvc != nil {
		enabled = h.systemSettingSvc.GetBool(
			c.Request.Context(),
			"client.workspace_binding_enabled",
			"XELORA_CLIENT_WORKSPACE_BINDING_ENABLED",
			false,
		)
	}

	c.JSON(http.StatusOK, ClientFeaturesResponse{
		Features: map[string]bool{"workspace_binding": enabled},
	})
}
