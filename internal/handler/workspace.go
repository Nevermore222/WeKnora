package handler

import (
	"errors"
	"mime"
	"net/http"
	"strings"

	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/Tencent/Xelora/internal/workspace"
	"github.com/gin-gonic/gin"
)

type workspaceMutationRequest struct {
	SessionID   string `json:"session_id"`
	WorkspaceID string `json:"workspace_id"`
	RootPath    string `json:"root_path"`
	Op          string `json:"op"`
	From        string `json:"from"`
	To          string `json:"to"`
	Token       string `json:"token"`
}

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
	if Edition == "personal" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "managed_workspace_disabled",
			"message": "Personal uses local folders; managed workspaces are disabled",
		})
		return
	}
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

const workspacePreviewMaxBytes int64 = 1 << 20

func (h *WorkspaceHandler) ListFiles(c *gin.Context) {
	files, err := h.service.ListFiles(c.Request.Context(), strings.TrimSpace(c.Param("id")), c.Query("path"))
	if err != nil {
		writeWorkspaceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"files": files})
}

func (h *WorkspaceHandler) PreviewFile(c *gin.Context) {
	preview, err := h.service.PreviewFile(c.Request.Context(), strings.TrimSpace(c.Param("id")), c.Query("path"), workspacePreviewMaxBytes)
	if err != nil {
		writeWorkspaceError(c, err)
		return
	}
	c.JSON(http.StatusOK, preview)
}

func (h *WorkspaceHandler) RawFile(c *gin.Context) {
	opened, err := h.service.OpenFile(c.Request.Context(), strings.TrimSpace(c.Param("id")), c.Query("path"))
	if err != nil {
		writeWorkspaceError(c, err)
		return
	}
	c.Header("Content-Type", opened.ContentType)
	c.File(opened.AbsolutePath)
}

func (h *WorkspaceHandler) DownloadFile(c *gin.Context) {
	opened, err := h.service.OpenFile(c.Request.Context(), strings.TrimSpace(c.Param("id")), c.Query("path"))
	if err != nil {
		writeWorkspaceError(c, err)
		return
	}
	c.Header("Content-Type", opened.ContentType)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": opened.Name}))
	c.File(opened.AbsolutePath)
}

// ProposeMutation creates a confirmable destructive filesystem change under a bound root.
func (h *WorkspaceHandler) ProposeMutation(c *gin.Context) {
	if Edition != "personal" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "mutation_personal_only", "message": "workspace mutations require Personal"})
		return
	}
	var req workspaceMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "message": "invalid mutation payload"})
		return
	}
	root, err := resolveMutationRoot(req.WorkspaceID, req.RootPath)
	if err != nil {
		writeWorkspaceError(c, err)
		return
	}
	proposal, err := workspace.DefaultMutationStore().Propose(workspace.MutationProposal{
		SessionID:   strings.TrimSpace(req.SessionID),
		WorkspaceID: strings.TrimSpace(req.WorkspaceID),
		RootPath:    root,
		Op:          req.Op,
		FromRel:     req.From,
		ToRel:       req.To,
	})
	if err != nil {
		writeWorkspaceError(c, err)
		return
	}
	c.JSON(http.StatusOK, proposal)
}

func (h *WorkspaceHandler) ApplyMutation(c *gin.Context) {
	var req workspaceMutationRequest
	_ = c.ShouldBindJSON(&req)
	token := strings.TrimSpace(req.Token)
	if token == "" {
		token = strings.TrimSpace(c.Query("token"))
	}
	if err := workspace.DefaultMutationStore().Apply(strings.TrimSpace(c.Param("id")), token); err != nil {
		writeMutationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "applied"})
}

func (h *WorkspaceHandler) RejectMutation(c *gin.Context) {
	var req workspaceMutationRequest
	_ = c.ShouldBindJSON(&req)
	token := strings.TrimSpace(req.Token)
	if token == "" {
		token = strings.TrimSpace(c.Query("token"))
	}
	if err := workspace.DefaultMutationStore().Reject(strings.TrimSpace(c.Param("id")), token); err != nil {
		writeMutationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "rejected"})
}

func resolveMutationRoot(workspaceID, rootPath string) (string, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	rootPath = strings.TrimSpace(rootPath)
	if root, ok := workspace.LookupLocalFolder(workspaceID); ok {
		if rootPath != "" && workspace.LocalWorkspaceID(rootPath) != workspaceID && root != rootPath {
			// Allow matching via normalize
			ent, err := workspace.BindLocalFolder(rootPath)
			if err != nil || ent.ID != workspaceID {
				return "", workspace.ErrNotFound
			}
			return ent.RootPath, nil
		}
		return root, nil
	}
	if rootPath == "" {
		return "", workspace.ErrNotFound
	}
	ent, err := workspace.BindLocalFolder(rootPath)
	if err != nil {
		return "", err
	}
	if workspaceID != "" && ent.ID != workspaceID {
		return "", workspace.ErrNotFound
	}
	workspace.RememberLocalFolder(ent.ID, ent.RootPath)
	return ent.RootPath, nil
}

func writeMutationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, workspace.ErrMutationNotFound):
		c.JSON(http.StatusNotFound, gin.H{"code": "mutation_not_found", "message": err.Error()})
	case errors.Is(err, workspace.ErrMutationExpired):
		c.JSON(http.StatusGone, gin.H{"code": "mutation_expired", "message": err.Error()})
	case errors.Is(err, workspace.ErrMutationToken):
		c.JSON(http.StatusForbidden, gin.H{"code": "mutation_token_invalid", "message": err.Error()})
	default:
		writeWorkspaceError(c, err)
	}
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
	case errors.Is(err, workspace.ErrInvalidName), errors.Is(err, workspace.ErrPathEscape), errors.Is(err, workspace.ErrMutationOp):
		c.JSON(http.StatusBadRequest, gin.H{"code": "workspace_invalid_name", "message": err.Error()})
	case errors.Is(err, workspace.ErrAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"code": "workspace_access_denied", "message": "workspace is not writable"})
	case errors.Is(err, workspace.ErrNotConfigured):
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "workspace_not_configured", "message": "workspace root is not configured"})
	case errors.Is(err, workspace.ErrFileTooLarge):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"code": "workspace_file_too_large", "message": "file is too large to preview"})
	case errors.Is(err, workspace.ErrUnsupportedPreview):
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"code": "workspace_preview_unsupported", "message": "file type is not supported for inline preview"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"code": "workspace_error", "message": "workspace operation failed"})
	}
}
