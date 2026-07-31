package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Tencent/Xelora/internal/agent/tools"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/workspace"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceMutateTool_RequiresPersonal(t *testing.T) {
	workspace.SetPersonalLocalFoldersEnabled(false)
	tool := tools.NewWorkspaceMutateTool()
	args, _ := json.Marshal(map[string]string{"op": "delete", "from": "a.txt"})
	res, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	require.False(t, res.Success)
	require.Contains(t, res.Error, "Personal")
}

func TestWorkspaceMutateTool_RequiresBinding(t *testing.T) {
	workspace.SetPersonalLocalFoldersEnabled(true)
	t.Cleanup(func() { workspace.SetPersonalLocalFoldersEnabled(false) })

	tool := tools.NewWorkspaceMutateTool().
		WithSessionWorkspaceResolver(func(string) *types.SessionWorkspaceBinding { return nil })
	args, _ := json.Marshal(map[string]string{"op": "delete", "from": "a.txt"})
	ctx := tools.WithToolExecContext(context.Background(), &tools.ToolExecContext{SessionID: "s1"})
	res, err := tool.Execute(ctx, args)
	require.NoError(t, err)
	require.False(t, res.Success)
	require.Contains(t, res.Error, "workspace_required")
}

func TestWorkspaceMutateTool_UserApprovesDelete(t *testing.T) {
	workspace.SetPersonalLocalFoldersEnabled(true)
	t.Cleanup(func() { workspace.SetPersonalLocalFoldersEnabled(false) })

	root := t.TempDir()
	target := filepath.Join(root, "a.txt")
	require.NoError(t, os.WriteFile(target, []byte("x"), 0o644))
	ent, err := workspace.BindLocalFolder(root)
	require.NoError(t, err)

	store := workspace.NewMutationStore(time.Minute)
	tool := tools.NewWorkspaceMutateTool().
		WithStore(store).
		WithSessionWorkspaceResolver(func(string) *types.SessionWorkspaceBinding {
			return &types.SessionWorkspaceBinding{
				WorkspaceID: ent.ID,
				RootPath:    ent.RootPath,
				Status:      types.SessionWorkspaceBindingStatusBound,
			}
		})

	args, _ := json.Marshal(map[string]string{"op": "delete", "from": "a.txt"})
	ctx := tools.WithToolExecContext(context.Background(), &tools.ToolExecContext{
		SessionID:   "sess-1",
		ApprovalCtx: context.Background(),
	})

	ch := make(chan *types.ToolResult, 1)
	go func() {
		res, _ := tool.Execute(ctx, args)
		ch <- res
	}()

	var proposal workspace.MutationProposal
	require.Eventually(t, func() bool {
		pending := store.ListPending()
		if len(pending) == 0 {
			return false
		}
		proposal = pending[0]
		return true
	}, 2*time.Second, 20*time.Millisecond)

	require.NoError(t, store.Apply(proposal.ID, proposal.Token))
	res := <-ch
	require.True(t, res.Success)
	_, err = os.Stat(target)
	require.True(t, os.IsNotExist(err))
}
