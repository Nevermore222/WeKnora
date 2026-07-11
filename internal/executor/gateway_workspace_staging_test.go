package executor

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Tencent/Xelora/internal/agent/skills"
	"github.com/Tencent/Xelora/internal/sandbox"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/stretchr/testify/require"
)

func testBoundWorkspace(root string) *types.SessionWorkspaceBinding {
	return &types.SessionWorkspaceBinding{
		WorkspaceID: "ws-test", RootPath: root, Status: types.SessionWorkspaceBindingStatusBound,
	}
}

func TestRunSkillScriptJobRequiresBoundWorkspace(t *testing.T) {
	basePath := t.TempDir()
	gateway := NewGatewayWithProviders(NewLocalProvider())
	executor := &fakeSkillExecutor{
		basePath: basePath,
		outcome: &skills.ScriptExecutionOutcome{
			Result:   &sandbox.ExecuteResult{ExitCode: 0, Duration: time.Millisecond},
			BasePath: basePath,
		},
	}

	_, err := gateway.RunSkillScriptJob(context.Background(), SkillJobRequest{
		Provider: LocalProviderName, SkillName: "writer", ScriptPath: "scripts/write.py",
	}, executor)

	require.ErrorContains(t, err, "workspace_required")
}

func TestRunBrowserTaskJobRequiresBoundWorkspace(t *testing.T) {
	provider := &fakeBrowserProvider{
		name:   "test-browser",
		status: ProviderStatusAvailable,
		result: BrowserTaskResult{Result: &sandbox.ExecuteResult{ExitCode: 0}},
	}
	gateway := newTestBrowserGateway(provider)

	_, err := gateway.RunBrowserTaskJob(context.Background(), BrowserJobRequest{
		Provider: "test-browser", URL: "https://example.com",
	}, &fakeSkillExecutor{basePath: t.TempDir()})

	require.ErrorContains(t, err, "workspace_required")
}

func TestStagePreparedInputsMovesRequestIntoWorkspace(t *testing.T) {
	workspaceRoot := t.TempDir()
	skillRoot := t.TempDir()
	requestPath := filepath.Join(skillRoot, "request.json")
	require.NoError(t, os.WriteFile(requestPath, []byte(`{"action":"write"}`), 0o600))
	prepared := &skills.PreparedScriptExecution{
		BasePath:               skillRoot,
		Args:                   []string{"request.json", "unchanged"},
		MaterializedInputPaths: []string{requestPath},
	}

	cleanup, err := stagePreparedInputs(prepared, workspaceRoot, "job-1")
	require.NoError(t, err)
	require.NotNil(t, cleanup)

	stagedRelative := filepath.Join(".xelora", "jobs", "job-1", "request.json")
	stagedAbsolute := filepath.Join(workspaceRoot, stagedRelative)
	require.Equal(t, stagedRelative, prepared.Args[0])
	require.Equal(t, "unchanged", prepared.Args[1])
	require.Equal(t, []string{stagedAbsolute}, prepared.MaterializedInputPaths)
	require.FileExists(t, stagedAbsolute)

	cleanup()
	require.NoFileExists(t, stagedAbsolute)
}

func TestSnapshotFilesIgnoresXeloraJobDirectory(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".xelora", "jobs", "job-1"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".xelora", "jobs", "job-1", "request.json"), []byte("{}"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "report.md"), []byte("ok"), 0o600))

	snapshot, err := snapshotFiles(root)
	require.NoError(t, err)
	require.Contains(t, snapshot, "report.md")
	require.NotContains(t, snapshot, filepath.Join(".xelora", "jobs", "job-1", "request.json"))
}

func TestIsWithinWorkspaceRootRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows symlink creation may require developer mode")
	}
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "outside-link")
	require.NoError(t, os.Symlink(outside, link))
	ctx := ConversationOutputContext{
		Mode: ConversationOutputModeBound, EffectiveRootDir: root, WriteAllowed: true,
	}

	require.False(t, ctx.IsWithinWorkspaceRoot(filepath.Join(link, "secret.txt")))
}
