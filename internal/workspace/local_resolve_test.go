package workspace_test

import (
	"context"
	"testing"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/workspace"
	"github.com/stretchr/testify/require"
)

func TestLocalRegistryResolve_LocalFolderIndex(t *testing.T) {
	dir := t.TempDir()
	ent, err := workspace.BindLocalFolder(dir)
	require.NoError(t, err)
	workspace.RememberLocalFolder(ent.ID, ent.RootPath)

	reg := workspace.NewLocalRegistry(t.TempDir())
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(1))
	ctx = context.WithValue(ctx, types.UserIDContextKey, "u1")

	got, err := reg.Resolve(ctx, ent.ID)
	require.NoError(t, err)
	require.Equal(t, ent.ID, got.ID)
	require.Equal(t, ent.RootPath, got.RootPath)
}
