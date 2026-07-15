package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/workspace"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type workspaceServiceStub struct {
	entries    []*workspace.Entry
	created    *workspace.Entry
	resolved   *workspace.Entry
	listErr    error
	createErr  error
	resolveErr error
	files      []*workspace.FileEntry
	fileErr    error
	preview    *workspace.FilePreview
	previewErr error
	opened     *workspace.FileOpenResult
	openErr    error
	input      workspace.CreateInput
}

func (s *workspaceServiceStub) List(context.Context) ([]*workspace.Entry, error) {
	return s.entries, s.listErr
}

func (s *workspaceServiceStub) Create(_ context.Context, input workspace.CreateInput) (*workspace.Entry, error) {
	s.input = input
	return s.created, s.createErr
}

func (s *workspaceServiceStub) Resolve(context.Context, string) (*workspace.Entry, error) {
	return s.resolved, s.resolveErr
}

func (s *workspaceServiceStub) ListFiles(context.Context, string, string) ([]*workspace.FileEntry, error) {
	return s.files, s.fileErr
}

func (s *workspaceServiceStub) PreviewFile(context.Context, string, string, int64) (*workspace.FilePreview, error) {
	return s.preview, s.previewErr
}

func (s *workspaceServiceStub) OpenFile(context.Context, string, string) (*workspace.FileOpenResult, error) {
	return s.opened, s.openErr
}

func workspaceHandlerContext(method, target string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(method, target, bytes.NewReader(body))
	ctx := context.WithValue(request.Context(), types.TenantIDContextKey, uint64(7))
	ctx = context.WithValue(ctx, types.UserIDContextKey, "user-1")
	c.Request = request.WithContext(ctx)
	return c, recorder
}

func TestWorkspaceHandlerListDoesNotExposeRootPath(t *testing.T) {
	stub := &workspaceServiceStub{entries: []*workspace.Entry{{
		ID: "ws-1", Name: "Reports", RelativePath: "Reports",
		RootPath: "/workspaces/Reports", Status: workspace.StatusAvailable,
	}}}
	handler := NewWorkspaceHandler(stub)
	c, recorder := workspaceHandlerContext(http.MethodGet, "/api/v1/workspaces", nil)

	handler.List(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"id":"ws-1"`)
	require.NotContains(t, recorder.Body.String(), "root_path")
	require.NotContains(t, recorder.Body.String(), "/workspaces")
}

func TestWorkspaceHandlerCreate(t *testing.T) {
	stub := &workspaceServiceStub{created: &workspace.Entry{
		ID: "ws-1", Name: "Reports", RelativePath: "Reports",
		RootPath: "/workspaces/Reports", Status: workspace.StatusAvailable,
	}}
	handler := NewWorkspaceHandler(stub)
	c, recorder := workspaceHandlerContext(http.MethodPost, "/api/v1/workspaces", []byte(`{"name":"Reports"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Equal(t, "Reports", stub.input.Name)
	require.NotContains(t, recorder.Body.String(), "root_path")
}

func TestWorkspaceHandlerGetMapsNotFound(t *testing.T) {
	stub := &workspaceServiceStub{resolveErr: workspace.ErrNotFound}
	handler := NewWorkspaceHandler(stub)
	c, recorder := workspaceHandlerContext(http.MethodGet, "/api/v1/workspaces/missing", nil)
	c.Params = gin.Params{{Key: "id", Value: "missing"}}

	handler.Get(c)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"code":"workspace_not_found"`)
}

func TestWorkspaceHandlerCreateMapsValidationErrors(t *testing.T) {
	stub := &workspaceServiceStub{createErr: errors.Join(workspace.ErrInvalidName, errors.New("bad name"))}
	handler := NewWorkspaceHandler(stub)
	c, recorder := workspaceHandlerContext(http.MethodPost, "/api/v1/workspaces", []byte(`{"name":"../bad"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"code":"workspace_invalid_name"`)
}

func TestWorkspaceHandlerPreviewFileRejectsTraversal(t *testing.T) {
	stub := &workspaceServiceStub{previewErr: workspace.ErrPathEscape}
	handler := NewWorkspaceHandler(stub)
	c, recorder := workspaceHandlerContext(http.MethodGet, "/api/v1/workspaces/ws-1/files/preview?path=../secret.txt", nil)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}

	handler.PreviewFile(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "workspace_invalid_name")
}

func TestWorkspaceHandlerDownloadSetsAttachment(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "summary.md")
	require.NoError(t, os.WriteFile(path, []byte("# Summary"), 0o644))
	stub := &workspaceServiceStub{opened: &workspace.FileOpenResult{
		Name: "summary.md", RelativePath: "summary.md", AbsolutePath: path,
		ContentType: "text/markdown; charset=utf-8", Size: 9,
	}}
	handler := NewWorkspaceHandler(stub)
	c, recorder := workspaceHandlerContext(http.MethodGet, "/api/v1/workspaces/ws-1/files/download?path=summary.md", nil)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}

	handler.DownloadFile(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Disposition"), "attachment")
}
