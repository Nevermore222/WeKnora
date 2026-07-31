package workspace_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tencent/Xelora/internal/workspace"
	"github.com/stretchr/testify/require"
)

func TestLocalWorkspaceID_Stable(t *testing.T) {
	a := workspace.LocalWorkspaceID(`D:\proj\Foo`)
	b := workspace.LocalWorkspaceID(`D:\proj\Foo`)
	require.True(t, strings.HasPrefix(a, "local:"))
	require.Equal(t, a, b)
}

func TestLocalWorkspaceID_Normalizes(t *testing.T) {
	id1 := workspace.LocalWorkspaceID(filepath.Clean(`D:\proj\Foo`))
	id2 := workspace.LocalWorkspaceID(filepath.Clean(`D:\proj\Foo\`))
	require.Equal(t, id1, id2)
}

func TestLocalWorkspaceName_Basename(t *testing.T) {
	require.Equal(t, "Foo", workspace.LocalWorkspaceName(filepath.Clean(`D:\proj\Foo`)))
}

func TestIsLocalWorkspaceID(t *testing.T) {
	require.True(t, workspace.IsLocalWorkspaceID("local:abc"))
	require.False(t, workspace.IsLocalWorkspaceID("uuid-style-id"))
}
