package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/Xelora/internal/application/repository"
	"github.com/Tencent/Xelora/internal/application/service"
	apperrors "github.com/Tencent/Xelora/internal/errors"
	"github.com/Tencent/Xelora/internal/logger"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
)

type OrganizationNodeHandler struct {
	service interfaces.OrganizationNodeService
}

func NewOrganizationNodeHandler(s interfaces.OrganizationNodeService) *OrganizationNodeHandler {
	return &OrganizationNodeHandler{service: s}
}

func (h *OrganizationNodeHandler) ListTree(c *gin.Context) {
	nodes, err := h.service.ListTree(c.Request.Context(), c.GetUint64(types.TenantIDContextKey.String()))
	if err != nil {
		logger.Errorf(c.Request.Context(), "list organization node tree: %v", err)
		c.Error(apperrors.NewInternalServerError("failed to list organization nodes"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": nodes})
}

func (h *OrganizationNodeHandler) Resources(c *gin.Context) {
	resources, err := h.service.ListResources(c.Request.Context(), c.GetUint64(types.TenantIDContextKey.String()), strings.TrimSpace(c.Param("id")))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resources})
}

func (h *OrganizationNodeHandler) AssignResource(c *gin.Context) {
	var req types.AssignOrganizationNodeResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid resource assignment"))
		return
	}
	err := h.service.AssignResource(c.Request.Context(), c.GetUint64(types.TenantIDContextKey.String()), strings.TrimSpace(c.Param("id")), &req)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *OrganizationNodeHandler) Create(c *gin.Context) {
	var req types.CreateOrganizationNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid organization node"))
		return
	}
	node, err := h.service.Create(c.Request.Context(), c.GetUint64(types.TenantIDContextKey.String()), &req)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": node})
}

func (h *OrganizationNodeHandler) Update(c *gin.Context) {
	var req types.UpdateOrganizationNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid organization node"))
		return
	}
	node, err := h.service.UpdateName(c.Request.Context(), c.GetUint64(types.TenantIDContextKey.String()), strings.TrimSpace(c.Param("id")), &req)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": node})
}

func (h *OrganizationNodeHandler) Delete(c *gin.Context) {
	err := h.service.Delete(c.Request.Context(), c.GetUint64(types.TenantIDContextKey.String()), strings.TrimSpace(c.Param("id")))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *OrganizationNodeHandler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrOrganizationNodeNotFound):
		c.Error(apperrors.NewNotFoundError("organization node not found"))
	case errors.Is(err, repository.ErrOrganizationResourceNotFound):
		c.Error(apperrors.NewNotFoundError("organization resource not found"))
	case errors.Is(err, repository.ErrOrganizationNodeHasChildren), errors.Is(err, service.ErrOrganizationNodeInvalid), errors.Is(err, service.ErrOrganizationNodeDepth), errors.Is(err, service.ErrOrganizationNodeName), errors.Is(err, service.ErrOrganizationNodeDuplicate):
		c.Error(apperrors.NewValidationError(err.Error()))
	default:
		logger.Errorf(c.Request.Context(), "organization node operation: %v", err)
		c.Error(apperrors.NewInternalServerError("organization node operation failed"))
	}
}
