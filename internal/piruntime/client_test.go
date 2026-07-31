package piruntime_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Tencent/Xelora/internal/piruntime"
	"github.com/stretchr/testify/require"
)

func TestClient_CallEcho(t *testing.T) {
	goBin, err := exec.LookPath("go")
	require.NoError(t, err)

	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	sidecar := filepath.Join(repoRoot, "internal", "piruntime", "testdata", "fake_sidecar.go")
	_, err = os.Stat(sidecar)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cli, err := piruntime.Start(ctx, piruntime.StartOptions{
		Command: goBin,
		Args:    []string{"run", sidecar},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })

	require.NoError(t, cli.Handshake(ctx))

	root := t.TempDir()
	resp, err := cli.Call(ctx, piruntime.Request{
		ProtocolVersion: piruntime.ProtocolVersion,
		CorrelationID:   "1",
		ToolName:        "workspace_read",
		Args:            json.RawMessage(`{"path":"a.txt"}`),
		Cwd:             root,
		TimeoutMs:       3000,
	})
	require.NoError(t, err)
	require.True(t, resp.OK)
	require.Contains(t, resp.Content, "a.txt")
	require.Contains(t, resp.Content, "workspace_read")
}
