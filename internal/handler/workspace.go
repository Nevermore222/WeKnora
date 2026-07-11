package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/Tencent/Xelora/internal/workspace"
	"github.com/gin-gonic/gin"
)

type WorkspaceHandler struct {
	service interfaces.WorkspaceService
}

func NewWorkspaceHandler(service interfaces.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{service: service}
}

func (h *WorkspaceHandler) List(c *gin.Context) {
	entries, err := h.service.List(c.Request.Context())
	if err != nil {
		writeWorkspaceError(c, err)
		return
	}
	response := make([]*workspace.Entry, 0, len(entries))
	for _, entry := range entries {
		response = append(response, workspaceResponseEntry(entry))
	}
	c.JSON(http.StatusOK, response)
}

func (h *WorkspaceHandler) Create(c *gin.Context) {
	var input workspace.CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "message": "workspace name is required"})
		return
	}
	entry, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		writeWorkspaceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, workspaceResponseEntry(entry))
}

func (h *WorkspaceHandler) Get(c *gin.Context) {
	entry, err := h.service.Resolve(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if err != nil {
		writeWorkspaceError(c, err)
		return
	}
	c.JSON(http.StatusOK, workspaceResponseEntry(entry))
}

func workspaceResponseEntry(entry *workspace.Entry) *workspace.Entry {
	if entry == nil {
		return nil
	}
	return &workspace.Entry{
		ID:           entry.ID,
		Name:         entry.Name,
		RelativePath: entry.RelativePath,
		Status:       entry.Status,
	}
}

func writeWorkspaceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, workspace.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"code": "workspace_not_found", "message": "workspace not found"})
	case errors.Is(err, workspace.ErrAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"code": "workspace_already_exists", "message": "workspace already exists"})
	case errors.Is(err, workspace.ErrInvalidName), errors.Is(err, workspace.ErrPathEscape):
		c.JSON(http.StatusBadRequest, gin.H{"code": "workspace_invalid_name", "message": err.Error()})
	case errors.Is(err, workspace.ErrAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"code": "workspace_access_denied", "message": "workspace is not writable"})
	case errors.Is(err, workspace.ErrNotConfigured):
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "workspace_not_configured", "message": "workspace root is not configured"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"code": "workspace_error", "message": "workspace operation failed"})
	}
}
