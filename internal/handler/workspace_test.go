package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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
