package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tencent/Xelora/internal/workspace"
	"github.com/stretchr/testify/require"
)

func TestBindLocalFolder_OK(t *testing.T) {
	dir := t.TempDir()
	ent, err := workspace.BindLocalFolder(dir)
	require.NoError(t, err)
	require.NotNil(t, ent)
	require.Equal(t, workspace.StatusAvailable, ent.Status)
	require.True(t, strings.HasPrefix(ent.ID, "local:"))
	require.Equal(t, filepath.Base(dir), ent.Name)
	require.Equal(t, filepath.Clean(dir), ent.RootPath)
}

func TestBindLocalFolder_Missing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	ent, err := workspace.BindLocalFolder(missing)
	require.ErrorIs(t, err, workspace.ErrNotFound)
	require.NotNil(t, ent)
	require.Equal(t, workspace.StatusMissing, ent.Status)
	require.True(t, strings.HasPrefix(ent.ID, "local:"))
}

func TestBindLocalFolder_NotDirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	ent, err := workspace.BindLocalFolder(file)
	require.Error(t, err)
	require.Nil(t, ent)
}
