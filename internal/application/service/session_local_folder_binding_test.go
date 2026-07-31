package service

import (
	"context"
	"strings"
	"testing"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/workspace"
	"github.com/stretchr/testify/require"
)

func TestNormalizeWorkspaceBinding_PersonalLocalFolder(t *testing.T) {
	workspace.SetPersonalLocalFoldersEnabled(true)
	t.Cleanup(func() { workspace.SetPersonalLocalFoldersEnabled(false) })

	dir := t.TempDir()
	svc := &sessionService{}
	out, err := svc.normalizeWorkspaceBinding(context.Background(), "user-1", &types.SessionWorkspaceBinding{
		RootPath: dir,
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Equal(t, types.SessionWorkspaceBindingKindLocalFolder, out.BindingKind)
	require.True(t, strings.HasPrefix(out.WorkspaceID, "local:"))
	require.Equal(t, types.SessionWorkspaceBindingStatusBound, out.Status)
	require.Equal(t, dir, out.RootPath)

	stored, ok := workspace.LookupLocalFolder(out.WorkspaceID)
	require.True(t, ok)
	require.Equal(t, dir, stored)
}

func TestNormalizeWorkspaceBinding_PersonalRequiresRootPath(t *testing.T) {
	workspace.SetPersonalLocalFoldersEnabled(true)
	t.Cleanup(func() { workspace.SetPersonalLocalFoldersEnabled(false) })

	svc := &sessionService{}
	out, err := svc.normalizeWorkspaceBinding(context.Background(), "user-1", &types.SessionWorkspaceBinding{
		WorkspaceID: "legacy-managed-id",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "binding_invalid")
	require.Equal(t, types.SessionWorkspaceBindingStatusInvalid, out.Status)
}

func TestNormalizeWorkspaceBinding_NonPersonalRejectsRootPath(t *testing.T) {
	workspace.SetPersonalLocalFoldersEnabled(false)
	svc := &sessionService{}
	_, err := svc.normalizeWorkspaceBinding(context.Background(), "user-1", &types.SessionWorkspaceBinding{
		RootPath: t.TempDir(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "root_path is only supported on Personal")
}
