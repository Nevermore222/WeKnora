package tools_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Tencent/Xelora/internal/agent/tools"
	"github.com/Tencent/Xelora/internal/piruntime"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/workspace"
	"github.com/stretchr/testify/require"
)

type stubCaller struct {
	last piruntime.Request
}

func (s *stubCaller) Handshake(context.Context) error { return nil }
func (s *stubCaller) Close() error                    { return nil }
func (s *stubCaller) Call(_ context.Context, req piruntime.Request) (piruntime.Response, error) {
	s.last = req
	return piruntime.Response{OK: true, Content: "listed"}, nil
}

func TestWorkspaceListTool_UsesFacadeCwd(t *testing.T) {
	workspace.SetPersonalLocalFoldersEnabled(true)
	t.Cleanup(func() { workspace.SetPersonalLocalFoldersEnabled(false) })

	root := t.TempDir()
	stub := &stubCaller{}
	facade := piruntime.NewFacade(stub, nil)
	tool := tools.NewWorkspaceListTool().
		WithFacade(facade).
		WithSessionWorkspaceResolver(func(string) *types.SessionWorkspaceBinding {
			return &types.SessionWorkspaceBinding{
				Status:   types.SessionWorkspaceBindingStatusBound,
				RootPath: root,
			}
		})

	ctx := tools.WithToolExecContext(context.Background(), &tools.ToolExecContext{SessionID: "s1"})
	res, err := tool.Execute(ctx, json.RawMessage(`{"path":"."}`))
	require.NoError(t, err)
	require.True(t, res.Success)
	require.Equal(t, root, stub.last.Cwd)
	require.Equal(t, tools.ToolWorkspaceList, stub.last.ToolName)
}
