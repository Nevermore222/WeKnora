package executor

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/Xelora/internal/sandbox"
	"github.com/Tencent/Xelora/internal/types"
)

type fakeBrowserProvider struct {
	name    string
	status  string
	result  BrowserTaskResult
	execErr error
}

func (f *fakeBrowserProvider) Name() string { return f.name }

func (f *fakeBrowserProvider) Capability(ctx context.Context) ProviderCapability {
	return ProviderCapability{
		Provider:                 f.name,
		Status:                   f.status,
		SupportsSessionWorkspace: true,
		LastCheckedAt:            time.Now(),
	}
}

func (f *fakeBrowserProvider) ExecuteBrowserTask(ctx context.Context, req BrowserJobRequest, outputDir string, executor SkillExecutor) (*BrowserTaskResult, error) {
	if f.execErr != nil {
		return nil, f.execErr
	}
	return &f.result, nil
}

func newTestBrowserGateway(provider BrowserProvider) *Gateway {
	gateway := NewGatewayWithProviders(NewLocalProvider())
	gateway.RegisterBrowserProvider(provider)
	return gateway
}

func TestGatewayRegistersBrowserProvider(t *testing.T) {
	gateway := NewGateway()
	provider, err := selectBrowserProvider("", gateway.browserProviders)
	if err != nil {
		t.Fatalf("expected browser provider to be registered: %v", err)
	}
	if provider.Name() != ControlledDockerBrowserProviderName {
		t.Fatalf("expected provider %s, got %s", ControlledDockerBrowserProviderName, provider.Name())
	}
}

func TestRunBrowserTaskJobRejectsUnknownBrowserProvider(t *testing.T) {
	basePath := t.TempDir()
	gateway := NewGateway()
	exec := &fakeSkillExecutor{basePath: basePath}
	_, err := gateway.RunBrowserTaskJob(context.Background(), BrowserJobRequest{
		Provider: "missing-browser",
		URL:      "https://example.com",
	}, exec)
	if err == nil {
		t.Fatal("expected unknown browser provider error")
	}
	if got := err.Error(); got != "browser provider not configured: missing-browser" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestRunBrowserTaskJobDefaultsToScreenshot(t *testing.T) {
	basePath := t.TempDir()
	workspaceRoot := t.TempDir()
	bp := &fakeBrowserProvider{
		name:   "test-browser",
		status: ProviderStatusAvailable,
		result: BrowserTaskResult{
			Result: &sandbox.ExecuteResult{
				ExitCode: 0,
				Duration: 100 * time.Millisecond,
				Stdout:   "ok",
			},
		},
	}
	gateway := newTestBrowserGateway(bp)
	exec := &fakeSkillExecutor{basePath: basePath}
	execution, err := gateway.RunBrowserTaskJob(context.Background(), BrowserJobRequest{
		Provider:         "test-browser",
		SessionID:        "session-1",
		URL:              "https://example.com",
		WorkspaceBinding: testBoundWorkspace(workspaceRoot),
	}, exec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execution.Job.CaptureMode != BrowserCaptureScreenshot {
		t.Fatalf("expected default capture mode screenshot, got %s", execution.Job.CaptureMode)
	}
	if execution.Job.Status != JobStatusSucceeded {
		t.Fatalf("expected succeeded, got %s", execution.Job.Status)
	}
}

func TestRunBrowserTaskJobRoutesToBoundWorkspace(t *testing.T) {
	skillBase := t.TempDir()
	workspaceRoot := t.TempDir()
	bp := &fakeBrowserProvider{
		name:   "test-browser",
		status: ProviderStatusAvailable,
		result: BrowserTaskResult{
			Result: &sandbox.ExecuteResult{
				ExitCode: 0,
				Duration: 50 * time.Millisecond,
			},
		},
	}
	gateway := newTestBrowserGateway(bp)
	exec := &fakeSkillExecutor{basePath: skillBase}
	binding := &types.SessionWorkspaceBinding{
		WorkspaceID: "tenant:1",
		RootPath:    workspaceRoot,
		Status:      types.SessionWorkspaceBindingStatusBound,
	}
	execution, err := gateway.RunBrowserTaskJob(context.Background(), BrowserJobRequest{
		Provider:         "test-browser",
		SessionID:        "session-bound",
		URL:              "https://example.com",
		WorkspaceBinding: binding,
	}, exec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execution.Workspace.RootPath != workspaceRoot {
		t.Fatalf("expected workspace root %s, got %s", workspaceRoot, execution.Workspace.RootPath)
	}
}

func TestRunBrowserTaskJobUnboundRequiresWorkspace(t *testing.T) {
	skillBase := t.TempDir()
	bp := &fakeBrowserProvider{
		name:   "test-browser",
		status: ProviderStatusAvailable,
		result: BrowserTaskResult{
			Result: &sandbox.ExecuteResult{
				ExitCode: 0,
				Duration: 30 * time.Millisecond,
			},
		},
	}
	gateway := newTestBrowserGateway(bp)
	exec := &fakeSkillExecutor{basePath: skillBase}
	_, err := gateway.RunBrowserTaskJob(context.Background(), BrowserJobRequest{
		Provider:  "test-browser",
		SessionID: "session-unbound",
		URL:       "https://example.com",
	}, exec)
	if err == nil || err.Error() != "workspace_required: bind a workspace before running file tools" {
		t.Fatalf("expected workspace_required, got %v", err)
	}
}

func TestRunBrowserTaskJobAccessDeniedBindingIsBlocked(t *testing.T) {
	skillBase := t.TempDir()
	bp := &fakeBrowserProvider{
		name:   "test-browser",
		status: ProviderStatusAvailable,
		result: BrowserTaskResult{
			Result: &sandbox.ExecuteResult{
				ExitCode: 0,
				Duration: 20 * time.Millisecond,
			},
		},
	}
	gateway := newTestBrowserGateway(bp)
	exec := &fakeSkillExecutor{basePath: skillBase}
	binding := &types.SessionWorkspaceBinding{
		WorkspaceID:       "tenant:1",
		Status:            types.SessionWorkspaceBindingStatusAccessDenied,
		ValidationMessage: "access revoked",
	}
	_, err := gateway.RunBrowserTaskJob(context.Background(), BrowserJobRequest{
		Provider:         "test-browser",
		SessionID:        "session-denied",
		URL:              "https://example.com",
		WorkspaceBinding: binding,
	}, exec)
	if err == nil || err.Error() != "access_denied: access revoked" {
		t.Fatalf("expected access_denied, got %v", err)
	}
}

func TestRunBrowserTaskJobFailedResult(t *testing.T) {
	basePath := t.TempDir()
	workspaceRoot := t.TempDir()
	bp := &fakeBrowserProvider{
		name:   "test-browser",
		status: ProviderStatusAvailable,
		result: BrowserTaskResult{
			Result: &sandbox.ExecuteResult{
				ExitCode: 1,
				Duration: 10 * time.Millisecond,
				Stderr:   "page not found",
				Error:    "navigation error",
			},
		},
	}
	gateway := newTestBrowserGateway(bp)
	exec := &fakeSkillExecutor{basePath: basePath}
	execution, err := gateway.RunBrowserTaskJob(context.Background(), BrowserJobRequest{
		Provider:         "test-browser",
		URL:              "https://example.com/notfound",
		WorkspaceBinding: testBoundWorkspace(workspaceRoot),
	}, exec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execution.Job.Status != JobStatusFailed {
		t.Fatalf("expected failed status, got %s", execution.Job.Status)
	}
	if execution.Job.Error != "navigation error" {
		t.Fatalf("expected error 'navigation error', got %s", execution.Job.Error)
	}
}
