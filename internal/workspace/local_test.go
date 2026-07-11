package workspace

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/stretchr/testify/require"
)

func workspaceTestContext(tenantID uint64, userID string) context.Context {
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, tenantID)
	return context.WithValue(ctx, types.UserIDContextKey, userID)
}

func TestLocalRegistryCreateListResolve(t *testing.T) {
	root := t.TempDir()
	registry := NewLocalRegistry(root)
	ctx := workspaceTestContext(7, "user-1")

	created, err := registry.Create(ctx, CreateInput{Name: "Quarterly Review"})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, "Quarterly Review", created.Name)
	require.Equal(t, "Quarterly Review", created.RelativePath)
	require.Equal(t, StatusAvailable, created.Status)
	require.DirExists(t, filepath.Join(root, created.RelativePath))

	listed, err := registry.List(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Empty(t, listed[0].RootPath)

	resolved, err := registry.Resolve(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, created.RelativePath), resolved.RootPath)
	require.Equal(t, StatusAvailable, resolved.Status)
	require.FileExists(t, filepath.Join(root, registryFileName))
}

func TestLocalRegistryScopesEntriesByTenantAndUser(t *testing.T) {
	registry := NewLocalRegistry(t.TempDir())
	owner := workspaceTestContext(7, "user-1")
	otherUser := workspaceTestContext(7, "user-2")
	otherTenant := workspaceTestContext(8, "user-1")

	created, err := registry.Create(owner, CreateInput{Name: "Private"})
	require.NoError(t, err)

	for _, ctx := range []context.Context{otherUser, otherTenant} {
		listed, listErr := registry.List(ctx)
		require.NoError(t, listErr)
		require.Empty(t, listed)
		_, resolveErr := registry.Resolve(ctx, created.ID)
		require.ErrorIs(t, resolveErr, ErrNotFound)
	}
}

func TestLocalRegistryRejectsInvalidNamesAndDuplicates(t *testing.T) {
	registry := NewLocalRegistry(t.TempDir())
	ctx := workspaceTestContext(7, "user-1")

	for _, name := range []string{"", ".", "..", "../outside", "nested/folder", `nested\folder`, "/absolute"} {
		_, err := registry.Create(ctx, CreateInput{Name: name})
		require.Error(t, err, name)
	}

	_, err := registry.Create(ctx, CreateInput{Name: "Reports"})
	require.NoError(t, err)
	_, err = registry.Create(ctx, CreateInput{Name: "Reports"})
	require.ErrorIs(t, err, ErrAlreadyExists)
}

func TestLocalRegistryRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows symlink creation may require developer mode")
	}

	root := t.TempDir()
	outside := t.TempDir()
	registry := NewLocalRegistry(root)
	ctx := workspaceTestContext(7, "user-1")
	created, err := registry.Create(ctx, CreateInput{Name: "Reports"})
	require.NoError(t, err)

	workspacePath := filepath.Join(root, created.RelativePath)
	require.NoError(t, os.Remove(workspacePath))
	require.NoError(t, os.Symlink(outside, workspacePath))

	_, err = registry.Resolve(ctx, created.ID)
	require.ErrorIs(t, err, ErrPathEscape)
}

func TestLocalRegistryReportsMissingDirectory(t *testing.T) {
	root := t.TempDir()
	registry := NewLocalRegistry(root)
	ctx := workspaceTestContext(7, "user-1")
	created, err := registry.Create(ctx, CreateInput{Name: "Reports"})
	require.NoError(t, err)
	require.NoError(t, os.Remove(filepath.Join(root, created.RelativePath)))

	listed, err := registry.List(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, StatusMissing, listed[0].Status)

	_, err = registry.Resolve(ctx, created.ID)
	require.ErrorIs(t, err, ErrNotFound)
}
