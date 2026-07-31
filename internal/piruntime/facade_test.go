package piruntime_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Tencent/Xelora/internal/piruntime"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/workspace"
	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	last piruntime.Request
}

func (f *fakeClient) Handshake(context.Context) error { return nil }
func (f *fakeClient) Close() error                    { return nil }
func (f *fakeClient) Call(_ context.Context, req piruntime.Request) (piruntime.Response, error) {
	f.last = req
	return piruntime.Response{OK: true, Content: "ok"}, nil
}

func TestFacade_NonPersonalUsesFallback(t *testing.T) {
	workspace.SetPersonalLocalFoldersEnabled(false)
	t.Cleanup(func() { workspace.SetPersonalLocalFoldersEnabled(false) })
	called := false
	f := piruntime.NewFacade(nil, func(ctx context.Context, tool string, args json.RawMessage) (*types.ToolResult, error) {
		called = true
		return &types.ToolResult{Success: true, Output: "go"}, nil
	})
	res, err := f.Execute(context.Background(), piruntime.ExecuteInput{
		SessionID: "s",
		ToolName:  "execute_skill_script",
		Args:      json.RawMessage(`{}`),
		Binding: &types.SessionWorkspaceBinding{
			Status:   types.SessionWorkspaceBindingStatusBound,
			RootPath: t.TempDir(),
		},
	})
	require.NoError(t, err)
	require.True(t, called)
	require.Equal(t, "go", res.Output)
}

func TestFacade_UnboundFails(t *testing.T) {
	workspace.SetPersonalLocalFoldersEnabled(true)
	t.Cleanup(func() { workspace.SetPersonalLocalFoldersEnabled(false) })
	f := piruntime.NewFacade(&fakeClient{}, nil)
	res, err := f.Execute(context.Background(), piruntime.ExecuteInput{
		ToolName: "workspace_read",
		Args:     json.RawMessage(`{"path":"a"}`),
		Binding:  nil,
	})
	require.NoError(t, err)
	require.False(t, res.Success)
	require.Contains(t, res.Error, "workspace_required")
}

func TestFacade_InjectsCwdFromBinding(t *testing.T) {
	workspace.SetPersonalLocalFoldersEnabled(true)
	t.Cleanup(func() { workspace.SetPersonalLocalFoldersEnabled(false) })
	root := t.TempDir()
	fc := &fakeClient{}
	f := piruntime.NewFacade(fc, nil)
	_, err := f.Execute(context.Background(), piruntime.ExecuteInput{
		SessionID: "s",
		ToolName:  "workspace_read",
		Args:      json.RawMessage(`{"path":"a.txt"}`),
		Binding: &types.SessionWorkspaceBinding{
			Status:   types.SessionWorkspaceBindingStatusBound,
			RootPath: root,
		},
	})
	require.NoError(t, err)
	require.Equal(t, root, fc.last.Cwd)
	require.Equal(t, piruntime.ProtocolVersion, fc.last.ProtocolVersion)
	require.Equal(t, "workspace_read", fc.last.ToolName)
}
