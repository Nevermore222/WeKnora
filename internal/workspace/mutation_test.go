package workspace_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Tencent/Xelora/internal/workspace"
	"github.com/stretchr/testify/require"
)

func TestApplyMutation_Delete(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "a.txt")
	require.NoError(t, os.WriteFile(target, []byte("x"), 0o644))
	store := workspace.NewMutationStore(time.Minute)
	p, err := store.Propose(workspace.MutationProposal{
		SessionID: "s1", WorkspaceID: "local:x", RootPath: root,
		Op: "delete", FromRel: "a.txt",
	})
	require.NoError(t, err)
	require.NoError(t, store.Apply(p.ID, p.Token))
	_, err = os.Stat(target)
	require.True(t, os.IsNotExist(err))
}

func TestApplyMutation_RejectsEscape(t *testing.T) {
	root := t.TempDir()
	store := workspace.NewMutationStore(time.Minute)
	_, err := store.Propose(workspace.MutationProposal{
		SessionID: "s1", WorkspaceID: "local:x", RootPath: root,
		Op: "delete", FromRel: "../outside.txt",
	})
	require.ErrorIs(t, err, workspace.ErrPathEscape)
}

func TestApplyMutation_Rename(t *testing.T) {
	root := t.TempDir()
	from := filepath.Join(root, "a.txt")
	require.NoError(t, os.WriteFile(from, []byte("x"), 0o644))
	store := workspace.NewMutationStore(time.Minute)
	p, err := store.Propose(workspace.MutationProposal{
		SessionID: "s1", WorkspaceID: "local:x", RootPath: root,
		Op: "rename", FromRel: "a.txt", ToRel: "b.txt",
	})
	require.NoError(t, err)
	require.NoError(t, store.Apply(p.ID, p.Token))
	_, err = os.Stat(from)
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(root, "b.txt"))
	require.NoError(t, err)
}

func TestRejectMutation_NoDiskChange(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "a.txt")
	require.NoError(t, os.WriteFile(target, []byte("x"), 0o644))
	store := workspace.NewMutationStore(time.Minute)
	p, err := store.Propose(workspace.MutationProposal{
		SessionID: "s1", WorkspaceID: "local:x", RootPath: root,
		Op: "delete", FromRel: "a.txt",
	})
	require.NoError(t, err)
	require.NoError(t, store.Reject(p.ID, p.Token))
	_, err = os.Stat(target)
	require.NoError(t, err)
}
