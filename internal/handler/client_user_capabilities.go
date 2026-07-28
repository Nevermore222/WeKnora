package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Tencent/Xelora/internal/logger"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/gin-gonic/gin"
)

type clientCapabilityDefaultsBody struct {
	CanCreateKnowledgeBase     bool `json:"can_create_knowledge_base"`
	CanCreateAgent             bool `json:"can_create_agent"`
	CanBindModel               bool `json:"can_bind_model"`
	TightenExistingClientUsers bool `json:"tighten_existing_client_users"`
}

type updateUserCapabilitiesBody struct {
	CanCreateKnowledgeBase bool `json:"can_create_knowledge_base"`
	CanCreateAgent         bool `json:"can_create_agent"`
	CanBindModel           bool `json:"can_bind_model"`
}

// GetClientCapabilityDefaults returns system defaults + tighten flag.
func (h *SystemHandler) GetClientCapabilityDefaults(c *gin.Context) {
	ctx := c.Request.Context()
	raw := h.systemSettingSvc.GetString(ctx, "auth.client_user_capability_defaults", "",
		`{"can_create_knowledge_base":false,"can_create_agent":false,"can_bind_model":false}`)
	var defs struct {
		CanCreateKnowledgeBase bool `json:"can_create_knowledge_base"`
		CanCreateAgent         bool `json:"can_create_agent"`
		CanBindModel           bool `json:"can_bind_model"`
	}
	_ = json.Unmarshal([]byte(raw), &defs)
	tighten := h.systemSettingSvc.GetString(ctx, "auth.tighten_existing_client_users", "", "false") == "true"
	c.JSON(http.StatusOK, gin.H{
		"defaults": gin.H{
			"can_create_knowledge_base": defs.CanCreateKnowledgeBase,
			"can_create_agent":          defs.CanCreateAgent,
			"can_bind_model":            defs.CanBindModel,
		},
		"tighten_existing_client_users": tighten,
	})
}

// UpdateClientCapabilityDefaults persists defaults and the tighten flag.
func (h *SystemHandler) UpdateClientCapabilityDefaults(c *gin.Context) {
	ctx := c.Request.Context()
	var body clientCapabilityDefaultsBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	payload, err := json.Marshal(map[string]bool{
		"can_create_knowledge_base": body.CanCreateKnowledgeBase,
		"can_create_agent":          body.CanCreateAgent,
		"can_bind_model":            body.CanBindModel,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encode failed"})
		return
	}
	if _, err := h.systemSettingSvc.Update(ctx, "auth.client_user_capability_defaults", string(payload)); err != nil {
		logger.Errorf(ctx, "UpdateClientCapabilityDefaults defaults: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tightenVal := "false"
	if body.TightenExistingClientUsers {
		tightenVal = "true"
	}
	if _, err := h.systemSettingSvc.Update(ctx, "auth.tighten_existing_client_users", tightenVal); err != nil {
		logger.Errorf(ctx, "UpdateClientCapabilityDefaults tighten flag: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.GetClientCapabilityDefaults(c)
}

// TightenExistingClientUsers applies defaults to non-overridden client users.
func (h *SystemHandler) TightenExistingClientUsers(c *gin.Context) {
	ctx := c.Request.Context()
	n, err := h.userSvc.TightenExistingClientUsers(ctx)
	if err != nil {
		logger.Errorf(ctx, "TightenExistingClientUsers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.emitAdminAudit(ctx, types.AuditActionClientCapabilitiesTightened, nil, map[string]any{
		"updated_count": n,
	})
	c.JSON(http.StatusOK, gin.H{"updated_count": n})
}

// ListClientUsers lists is_client_provisioned users.
func (h *SystemHandler) ListClientUsers(c *gin.Context) {
	ctx := c.Request.Context()
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	users, total, err := h.userSvc.ListClientProvisionedUsers(ctx, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	infos := make([]*types.UserInfo, 0, len(users))
	for _, u := range users {
		infos = append(infos, u.ToUserInfo())
	}
	c.JSON(http.StatusOK, gin.H{"users": infos, "total": total})
}

// ListAdminUsers lists all users for the User permissions settings page.
func (h *SystemHandler) ListAdminUsers(c *gin.Context) {
	ctx := c.Request.Context()
	q := c.Query("q")
	filter := types.AdminUserListFilter(c.DefaultQuery("filter", "all"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, total, err := h.userSvc.ListAdminUsers(ctx, q, filter, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rows := make([]gin.H, 0, len(items))
	for _, it := range items {
		u := it.User
		if u == nil {
			continue
		}
		rows = append(rows, gin.H{
			"id":                        u.ID,
			"username":                  u.Username,
			"email":                     u.Email,
			"is_system_admin":           u.IsSystemAdmin,
			"is_client_provisioned":     u.IsClientProvisioned,
			"can_create_knowledge_base": u.CanCreateKnowledgeBase,
			"can_create_agent":          u.CanCreateAgent,
			"can_bind_model":            u.CanBindModel,
			"home_tenant_id":            u.TenantID,
			"home_tenant_name":          it.HomeTenantName,
		})
	}
	c.JSON(http.StatusOK, gin.H{"users": rows, "total": total})
}

// GetUserCapabilities returns capability fields for one user.
func (h *SystemHandler) GetUserCapabilities(c *gin.Context) {
	ctx := c.Request.Context()
	user, err := h.userSvc.GetUserByID(ctx, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user.ToUserInfo())
}

// UpdateUserCapabilities sets capability flags (system admin; any user).
func (h *SystemHandler) UpdateUserCapabilities(c *gin.Context) {
	ctx := c.Request.Context()
	var body updateUserCapabilitiesBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	user, err := h.userSvc.UpdateUserCapabilities(ctx, c.Param("id"),
		body.CanCreateKnowledgeBase, body.CanCreateAgent, body.CanBindModel)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.emitAdminAudit(ctx, types.AuditActionClientCapabilitiesUpdated, user, map[string]any{
		"can_create_knowledge_base": body.CanCreateKnowledgeBase,
		"can_create_agent":          body.CanCreateAgent,
		"can_bind_model":            body.CanBindModel,
	})
	c.JSON(http.StatusOK, user.ToUserInfo())
}
