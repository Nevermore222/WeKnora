package handler

import (
	"net/http"
	"os"
	"strings"

	"github.com/Tencent/Xelora/internal/errors"
	"github.com/Tencent/Xelora/internal/logger"
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// SkillHandler handles skill-related HTTP requests
type SkillHandler struct {
	skillService interfaces.SkillService
}

// NewSkillHandler creates a new skill handler
func NewSkillHandler(skillService interfaces.SkillService) *SkillHandler {
	return &SkillHandler{
		skillService: skillService,
	}
}

// SkillInfoResponse represents the skill info returned to frontend
type SkillInfoResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SkillFileResponse represents a skill resource file returned to frontend.
type SkillFileResponse struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	IsScript bool   `json:"is_script"`
}

// ListSkills godoc
// @Summary      获取预装Skills列表
// @Description  获取所有预装的Agent Skills元数据
// @Tags         Skills
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Skills列表"
// @Failure      500  {object}  errors.AppError         "服务器错误"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /skills [get]
func (h *SkillHandler) ListSkills(c *gin.Context) {
	ctx := c.Request.Context()

	skillSummaries, err := h.skillService.ListSkillSummaries(ctx)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError("Failed to list skills: " + err.Error()))
		return
	}

	// skills_available: true only when sandbox is enabled (docker or local), so frontend can hide/disable Skills UI
	sandboxMode := os.Getenv("XELORA_SANDBOX_MODE")
	skillsAvailable := sandboxMode != "" && sandboxMode != "disabled"

	logger.Infof(ctx, "skills_available: %v, sandboxMode: %s", skillsAvailable, sandboxMode)

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"data":             skillSummaries,
		"skills_available": skillsAvailable,
	})
}

// GetSkill godoc
// @Summary      Get skill detail
// @Description  Get a preloaded skill's instructions and resource summaries
// @Tags         Skills
// @Accept       json
// @Produce      json
// @Param        name  path      string  true  "Skill name"
// @Success      200   {object}  map[string]interface{}  "Skill detail"
// @Failure      404   {object}  errors.AppError         "Skill not found"
// @Failure      500   {object}  errors.AppError         "Server error"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /skills/{name} [get]
func (h *SkillHandler) GetSkill(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("name")

	detail, err := h.skillService.GetSkillDetail(ctx, name)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		if strings.Contains(err.Error(), "not found") {
			c.Error(errors.NewNotFoundError("Skill not found: " + name))
			return
		}
		c.Error(errors.NewInternalServerError("Failed to get skill: " + err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// GetSkillFile godoc
// @Summary      Get skill file
// @Description  Get a resource file inside a preloaded skill directory
// @Tags         Skills
// @Accept       json
// @Produce      json
// @Param        name  path      string  true  "Skill name"
// @Param        path  path      string  true  "Relative file path"
// @Success      200   {object}  map[string]interface{}  "Skill file"
// @Failure      400   {object}  errors.AppError         "Invalid file path"
// @Failure      404   {object}  errors.AppError         "Skill or file not found"
// @Failure      500   {object}  errors.AppError         "Server error"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /skills/{name}/files/{path} [get]
func (h *SkillHandler) GetSkillFile(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("name")
	path := strings.TrimPrefix(c.Param("path"), "/")
	if path == "" {
		c.Error(errors.NewBadRequestError("Skill file path is required"))
		return
	}

	file, err := h.skillService.GetSkillFile(ctx, name, path)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		message := err.Error()
		switch {
		case strings.Contains(message, "invalid file path") || strings.Contains(message, "outside skill directory"):
			c.Error(errors.NewBadRequestError("Invalid skill file path"))
		case strings.Contains(message, "not found") || strings.Contains(message, "no such file"):
			c.Error(errors.NewNotFoundError("Skill file not found: " + path))
		default:
			c.Error(errors.NewInternalServerError("Failed to get skill file: " + message))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": SkillFileResponse{
			Path:     path,
			Content:  file.Content,
			IsScript: file.IsScript,
		},
	})
}
