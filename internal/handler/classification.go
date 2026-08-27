package handler

import (
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"net/http"
)

type ClassificationHandler struct {
	service interfaces.ClassificationService
}

func NewClassificationHandler(s interfaces.ClassificationService) *ClassificationHandler {
	return &ClassificationHandler{service: s}
}
func (h *ClassificationHandler) List(c *gin.Context) {
	resourceType := c.DefaultQuery("resource_type", "knowledge_base")
	v, e := h.service.List(c.Request.Context(), resourceType)
	if e != nil {
		c.Error(e)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": v})
}

type moduleRequest struct {
	Name      string `json:"name" binding:"required"`
	SortOrder int    `json:"sort_order"`
}

func (h *ClassificationHandler) CreateModule(c *gin.Context) {
	resourceType := c.DefaultQuery("resource_type", "knowledge_base")
	var r moduleRequest
	if c.ShouldBindJSON(&r) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	v, e := h.service.CreateModule(c.Request.Context(), resourceType, r.Name, r.SortOrder)
	if e != nil {
		c.Error(e)
		return
	}
	c.JSON(201, v)
}

type modulePatch struct {
	Name      *string `json:"name"`
	SortOrder *int    `json:"sort_order"`
	Enabled   *bool   `json:"enabled"`
}

func (h *ClassificationHandler) UpdateModule(c *gin.Context) {
	resourceType := c.DefaultQuery("resource_type", "knowledge_base")
	var r modulePatch
	if c.ShouldBindJSON(&r) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	v, e := h.service.UpdateModule(c.Request.Context(), resourceType, c.Param("id"), r.Name, r.SortOrder, r.Enabled)
	if e != nil {
		c.Error(e)
		return
	}
	c.JSON(200, v)
}
func (h *ClassificationHandler) DeleteModule(c *gin.Context) {
	resourceType := c.DefaultQuery("resource_type", "knowledge_base")
	if e := h.service.DeleteModule(c.Request.Context(), resourceType, c.Param("id")); e != nil {
		c.Error(e)
		return
	}
	c.Status(204)
}

type optionRequest struct {
	ModuleID  string  `json:"module_id" binding:"required"`
	ParentID  *string `json:"parent_id"`
	Name      string  `json:"name" binding:"required"`
	SortOrder int     `json:"sort_order"`
}

func (h *ClassificationHandler) CreateOption(c *gin.Context) {
	resourceType := c.DefaultQuery("resource_type", "knowledge_base")
	var r optionRequest
	if c.ShouldBindJSON(&r) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	v, e := h.service.CreateOption(c.Request.Context(), resourceType, r.ModuleID, r.ParentID, r.Name, r.SortOrder)
	if e != nil {
		c.Error(e)
		return
	}
	c.JSON(201, v)
}
func (h *ClassificationHandler) UpdateOption(c *gin.Context) {
	resourceType := c.DefaultQuery("resource_type", "knowledge_base")
	var r modulePatch
	if c.ShouldBindJSON(&r) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	v, e := h.service.UpdateOption(c.Request.Context(), resourceType, c.Param("id"), r.Name, r.SortOrder, r.Enabled)
	if e != nil {
		c.Error(e)
		return
	}
	c.JSON(200, v)
}
func (h *ClassificationHandler) DeleteOption(c *gin.Context) {
	resourceType := c.DefaultQuery("resource_type", "knowledge_base")
	if e := h.service.DeleteOption(c.Request.Context(), resourceType, c.Param("id")); e != nil {
		c.Error(e)
		return
	}
	c.Status(204)
}
