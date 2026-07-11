package executor

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Tencent/Xelora/internal/agent/skills"
	"github.com/Tencent/Xelora/internal/sandbox"
	"github.com/Tencent/Xelora/internal/types"
)

type fakeSkillExecutor struct {
	basePath string
	prepared *skills.PreparedScriptExecution
	outcome  *skills.ScriptExecutionOutcome
	err      error
}

func (f *fakeSkillExecutor) GetSkillBasePath(ctx context.Context, skillName string) (string, error) {
	return f.basePath, nil
}

func (f *fakeSkillExecutor) ExecuteScriptDetailed(ctx context.Context, skillName, scriptPath string, args []string, stdin string) (*skills.ScriptExecutionOutcome, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.outcome, nil
}

func (f *fakeSkillExecutor) PrepareScriptExecution(ctx context.Context, skillName, scriptPath string, args []string, stdin string) (*skills.PreparedScriptExecution, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.prepared != nil {
		return f.prepared, nil
	}
	return &skills.PreparedScriptExecution{
		BasePath:   f.basePath,
		ScriptPath: filepath.Join(f.basePath, scriptPath),
		Args:       append([]string(nil), args...),
		Stdin:      stdin,
	}, nil
}

func (f *fakeSkillExecutor) ExecutePreparedScript(ctx context.Context, prepared *skills.PreparedScriptExecution) (*skills.ScriptExecutionOutcome, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.outcome, nil
}

func TestDetectArtifactsIgnoresMaterializedInput(t *testing.T) {
	basePath := filepath.Clean("/tmp/skill")
	now := time.Now()
	pre := map[string]fileSnapshot{
		"existing.md": {
			RelativePath: "existing.md",
			AbsolutePath: filepath.Join(basePath, "existing.md"),
			Size:         5,
			ModifiedAt:   now,
		},
	}
	post := map[string]fileSnapshot{
		"generated.md": {
			RelativePath: "generated.md",
			AbsolutePath: filepath.Join(basePath, "generated.md"),
			Size:         9,
			ModifiedAt:   now.Add(time.Second),
		},
		"input.md": {
			RelativePath: "input.md",
			AbsolutePath: filepath.Join(basePath, "input.md"),
			Size:         10,
			ModifiedAt:   now.Add(time.Second),
		},
	}

	artifacts := detectArtifacts(basePath, "workspace-1", "job-1", "session-1", pre, post, []string{filepath.Join(basePath, "input.md")})
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	artifact := artifacts[0]
	if artifact.RelativePath != "generated.md" {
		t.Fatalf("expected generated.md artifact, got %s", artifact.RelativePath)
	}
	if artifact.WorkspaceID != "workspace-1" {
		t.Fatalf("expected workspace-1, got %s", artifact.WorkspaceID)
	}
	if artifact.Kind != ArtifactKindMarkdown {
		t.Fatalf("expected markdown kind, got %s", artifact.Kind)
	}
	if artifact.PreviewState != ArtifactPreviewAvailable {
		t.Fatalf("expected available preview, got %s", artifact.PreviewState)
	}
	if artifact.ChangeType != ArtifactChangeCreated {
		t.Fatalf("expected created change type, got %s", artifact.ChangeType)
	}
}

func TestBuildWorkspaceIDUsesSessionAndSkill(t *testing.T) {
	id := buildWorkspaceID("Session:123", "baoyu-format-markdown")
	want := "session-session-123-skill-baoyu-format-markdown"
	if id != want {
		t.Fatalf("expected %s, got %s", want, id)
	}
}

func TestRunSkillScriptJobDefaultsToLocalProvider(t *testing.T) {
	basePath := t.TempDir()
	workspaceRoot := t.TempDir()
	gateway := NewGateway()
	executor := &fakeSkillExecutor{
		basePath: basePath,
		outcome: &skills.ScriptExecutionOutcome{
			Result: &sandbox.ExecuteResult{
				ExitCode: 0,
				Duration: 25 * time.Millisecond,
				Stdout:   "ok",
			},
			BasePath:               basePath,
			MaterializedInputPaths: nil,
		},
	}

	execution, err := gateway.RunSkillScriptJob(context.Background(), SkillJobRequest{
		Provider:         LocalProviderName,
		SessionID:        "session-1",
		SkillName:        "demo-skill",
		ScriptPath:       "scripts/demo.py",
		WorkspaceBinding: testBoundWorkspace(workspaceRoot),
	}, executor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execution.Job.Provider != LocalProviderName {
		t.Fatalf("expected provider %s, got %s", LocalProviderName, execution.Job.Provider)
	}
	if execution.Workspace.Provider != LocalProviderName {
		t.Fatalf("expected workspace provider %s, got %s", LocalProviderName, execution.Workspace.Provider)
	}
	if execution.Provider.Provider != LocalProviderName {
		t.Fatalf("expected provider capability for %s, got %s", LocalProviderName, execution.Provider.Provider)
	}
	if execution.Provider.Status != ProviderStatusAvailable {
		t.Fatalf("expected provider status available, got %s", execution.Provider.Status)
	}
	if execution.Job.Status != JobStatusSucceeded {
		t.Fatalf("expected succeeded job, got %s", execution.Job.Status)
	}
	if execution.Job.WorkspaceID == "" {
		t.Fatal("expected workspace id to be populated")
	}
	if execution.Job.WorkspaceID != execution.Workspace.ID {
		t.Fatalf("expected workspace ids to match, got %s and %s", execution.Job.WorkspaceID, execution.Workspace.ID)
	}
	if execution.ArtifactDetected {
		t.Fatal("expected no artifacts for empty workspace execution")
	}
}

func TestSelectProviderDefaultsToControlledDocker(t *testing.T) {
	provider, err := selectProvider("", map[string]Provider{
		ControlledDockerProviderName: NewControlledDockerProvider(),
		LocalProviderName:            NewLocalProvider(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Name() != ControlledDockerProviderName {
		t.Fatalf("expected default provider %s, got %s", ControlledDockerProviderName, provider.Name())
	}
}

func TestGatewayRegistersControlledDockerProvider(t *testing.T) {
	gateway := NewGateway()
	provider, err := selectProvider(ControlledDockerProviderName, gateway.providers)
	if err != nil {
		t.Fatalf("expected controlled docker provider to be registered: %v", err)
	}
	if provider.Name() != ControlledDockerProviderName {
		t.Fatalf("expected provider %s, got %s", ControlledDockerProviderName, provider.Name())
	}
}

func TestGatewayRegistersOpenSandboxProvider(t *testing.T) {
	gateway := NewGateway()
	provider, err := selectProvider(OpenSandboxProviderName, gateway.providers)
	if err != nil {
		t.Fatalf("expected opensandbox provider to be registered: %v", err)
	}
	if provider.Name() != OpenSandboxProviderName {
		t.Fatalf("expected provider %s, got %s", OpenSandboxProviderName, provider.Name())
	}
}

func TestRunSkillScriptJobRejectsUnknownProvider(t *testing.T) {
	basePath := t.TempDir()
	gateway := NewGateway()
	executor := &fakeSkillExecutor{basePath: basePath}

	_, err := gateway.RunSkillScriptJob(context.Background(), SkillJobRequest{
		Provider:   "missing-provider",
		SkillName:  "demo-skill",
		ScriptPath: "scripts/demo.py",
	}, executor)
	if err == nil {
		t.Fatal("expected unknown provider error")
	}
	if got := err.Error(); got != "executor provider not configured: missing-provider" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestCubeSandboxProviderCapabilityUnavailableWithoutConfig(t *testing.T) {
	t.Setenv("E2B_API_URL", "")
	t.Setenv("E2B_API_KEY", "")
	t.Setenv("CUBE_TEMPLATE_ID", "")

	provider := NewCubeSandboxProvider()
	capability := provider.Capability(context.Background())
	if capability.Provider != CubeSandboxProviderName {
		t.Fatalf("expected %s provider, got %s", CubeSandboxProviderName, capability.Provider)
	}
	if capability.Status != ProviderStatusUnavailable {
		t.Fatalf("expected unavailable status, got %s", capability.Status)
	}
}

func TestOpenSandboxProviderCapabilityUnavailableWithoutConfig(t *testing.T) {
	t.Setenv("XELORA_OPENSANDBOX_BASE_URL", "")
	t.Setenv("XELORA_OPENSANDBOX_API_KEY", "")
	t.Setenv("XELORA_OPENSANDBOX_TEMPLATE_ID", "")

	provider := NewOpenSandboxProvider()
	capability := provider.Capability(context.Background())
	if capability.Provider != OpenSandboxProviderName {
		t.Fatalf("expected %s provider, got %s", OpenSandboxProviderName, capability.Provider)
	}
	if capability.Status != ProviderStatusUnavailable {
		t.Fatalf("expected unavailable status, got %s", capability.Status)
	}
}

func TestRunSkillScriptJobRoutesToBoundWorkspace(t *testing.T) {
	skillBase := t.TempDir()
	workspaceRoot := t.TempDir()

	gateway := NewGateway()
	executor := &fakeSkillExecutor{
		basePath: skillBase,
		prepared: &skills.PreparedScriptExecution{
			BasePath:   skillBase,
			ScriptPath: filepath.Join(skillBase, "scripts", "demo.py"),
		},
		outcome: &skills.ScriptExecutionOutcome{
			Result: &sandbox.ExecuteResult{
				ExitCode: 0,
				Duration: 10 * time.Millisecond,
				Stdout:   "ok",
			},
			BasePath: skillBase,
		},
	}

	binding := &types.SessionWorkspaceBinding{
		WorkspaceID: "tenant:1",
		RootPath:    workspaceRoot,
		Status:      types.SessionWorkspaceBindingStatusBound,
	}

	execution, err := gateway.RunSkillScriptJob(context.Background(), SkillJobRequest{
		Provider:         LocalProviderName,
		SessionID:        "session-bound",
		SkillName:        "demo-skill",
		ScriptPath:       "scripts/demo.py",
		WorkspaceBinding: binding,
	}, executor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execution.Job.Status != JobStatusSucceeded {
		t.Fatalf("expected succeeded, got %s", execution.Job.Status)
	}
	if execution.Workspace.RootPath != workspaceRoot {
		t.Fatalf("expected workspace root %s, got %s", workspaceRoot, execution.Workspace.RootPath)
	}
	if execution.Workspace.WorkingDir != workspaceRoot {
		t.Fatalf("expected working dir %s, got %s", workspaceRoot, execution.Workspace.WorkingDir)
	}
}

func TestRunSkillScriptJobUnboundRequiresWorkspace(t *testing.T) {
	skillBase := t.TempDir()

	gateway := NewGateway()
	executor := &fakeSkillExecutor{
		basePath: skillBase,
		outcome: &skills.ScriptExecutionOutcome{
			Result: &sandbox.ExecuteResult{
				ExitCode: 0,
				Duration: 5 * time.Millisecond,
			},
			BasePath: skillBase,
		},
	}

	_, err := gateway.RunSkillScriptJob(context.Background(), SkillJobRequest{
		Provider:   LocalProviderName,
		SessionID:  "session-unbound",
		SkillName:  "demo-skill",
		ScriptPath: "scripts/demo.py",
	}, executor)
	if err == nil || err.Error() != "workspace_required: bind a workspace before running file tools" {
		t.Fatalf("expected workspace_required, got %v", err)
	}
}

func TestRunSkillScriptJobBlockedBindingIsRejected(t *testing.T) {
	skillBase := t.TempDir()

	gateway := NewGateway()
	executor := &fakeSkillExecutor{
		basePath: skillBase,
		outcome: &skills.ScriptExecutionOutcome{
			Result: &sandbox.ExecuteResult{
				ExitCode: 0,
				Duration: 5 * time.Millisecond,
			},
			BasePath: skillBase,
		},
	}

	binding := &types.SessionWorkspaceBinding{
		WorkspaceID:       "tenant:1",
		Status:            types.SessionWorkspaceBindingStatusInvalid,
		ValidationMessage: "workspace no longer accessible",
	}

	_, err := gateway.RunSkillScriptJob(context.Background(), SkillJobRequest{
		Provider:         LocalProviderName,
		SessionID:        "session-blocked",
		SkillName:        "demo-skill",
		ScriptPath:       "scripts/demo.py",
		WorkspaceBinding: binding,
	}, executor)
	if err == nil || err.Error() != "binding_invalid: workspace no longer accessible" {
		t.Fatalf("expected binding_invalid, got %v", err)
	}
}

func TestResolveConversationOutputContextBound(t *testing.T) {
	ctx := ResolveConversationOutputContext("s1", &types.SessionWorkspaceBinding{
		WorkspaceID: "tenant:1",
		RootPath:    "/data/files/session-workspaces/tenant-1",
		Status:      types.SessionWorkspaceBindingStatusBound,
	})
	if ctx.Mode != ConversationOutputModeBound {
		t.Fatalf("expected bound mode, got %s", ctx.Mode)
	}
	if !ctx.WriteAllowed {
		t.Fatal("expected write allowed for bound context")
	}
	if ctx.EffectiveRootDir == "" {
		t.Fatal("expected non-empty effective root dir")
	}
}

func TestResolveConversationOutputContextUnbound(t *testing.T) {
	ctx := ResolveConversationOutputContext("s1", nil)
	if ctx.Mode != ConversationOutputModeUnbound {
		t.Fatalf("expected unbound mode, got %s", ctx.Mode)
	}
	if ctx.WriteAllowed {
		t.Fatal("expected write not allowed for unbound context")
	}
}

func TestIsWithinWorkspaceRoot(t *testing.T) {
	ctx := ConversationOutputContext{
		Mode:             ConversationOutputModeBound,
		EffectiveRootDir: "/data/files/session-workspaces/tenant-1",
	}
	if !ctx.IsWithinWorkspaceRoot("/data/files/session-workspaces/tenant-1/output.md") {
		t.Fatal("expected path within root to pass")
	}
	if ctx.IsWithinWorkspaceRoot("/data/files/session-workspaces/tenant-2/output.md") {
		t.Fatal("expected path outside root to fail")
	}
}

func TestResolveConversationOutputContextAccessDenied(t *testing.T) {
	ctx := ResolveConversationOutputContext("s1", &types.SessionWorkspaceBinding{
		WorkspaceID:       "tenant:1",
		Status:            types.SessionWorkspaceBindingStatusAccessDenied,
		ValidationMessage: "access revoked",
	})
	if ctx.Mode != ConversationOutputModeBound {
		t.Fatalf("expected bound mode, got %s", ctx.Mode)
	}
	if ctx.WriteAllowed {
		t.Fatal("expected write not allowed for access_denied context")
	}
	if ctx.FailureCode != ConversationOutputFailureAccessDenied {
		t.Fatalf("expected access_denied failure code, got %s", ctx.FailureCode)
	}
	if ctx.FailureMessage != "access revoked" {
		t.Fatalf("expected failure message 'access revoked', got %s", ctx.FailureMessage)
	}
}

func TestResolveConversationOutputContextArchived(t *testing.T) {
	ctx := ResolveConversationOutputContext("s1", &types.SessionWorkspaceBinding{
		WorkspaceID:       "tenant:1",
		Status:            types.SessionWorkspaceBindingStatusArchived,
		ValidationMessage: "workspace archived",
	})
	if ctx.WriteAllowed {
		t.Fatal("expected write not allowed for archived context")
	}
	if ctx.FailureCode != ConversationOutputFailureBindingInvalid {
		t.Fatalf("expected binding_invalid failure code, got %s", ctx.FailureCode)
	}
}

func TestRunSkillScriptJobAccessDeniedBindingIsRejected(t *testing.T) {
	skillBase := t.TempDir()

	gateway := NewGateway()
	executor := &fakeSkillExecutor{
		basePath: skillBase,
		outcome: &skills.ScriptExecutionOutcome{
			Result: &sandbox.ExecuteResult{
				ExitCode: 0,
				Duration: 5 * time.Millisecond,
			},
			BasePath: skillBase,
		},
	}

	binding := &types.SessionWorkspaceBinding{
		WorkspaceID:       "tenant:1",
		Status:            types.SessionWorkspaceBindingStatusAccessDenied,
		ValidationMessage: "access revoked",
	}

	_, err := gateway.RunSkillScriptJob(context.Background(), SkillJobRequest{
		Provider:         LocalProviderName,
		SessionID:        "session-denied",
		SkillName:        "demo-skill",
		ScriptPath:       "scripts/demo.py",
		WorkspaceBinding: binding,
	}, executor)
	if err == nil || err.Error() != "access_denied: access revoked" {
		t.Fatalf("expected access_denied, got %v", err)
	}
}

func TestIsWithinWorkspaceRootPathEscapeDetection(t *testing.T) {
	ctx := ConversationOutputContext{
		Mode:             ConversationOutputModeBound,
		EffectiveRootDir: "/data/files/session-workspaces/tenant-1",
	}
	escapePath := "/data/files/session-workspaces/tenant-10/secret.md"
	if ctx.IsWithinWorkspaceRoot(escapePath) {
		t.Fatalf("expected sibling path escape to be detected: %s", escapePath)
	}
	traversalPath := filepath.Clean("/data/files/session-workspaces/tenant-1/../../../etc/passwd")
	if ctx.IsWithinWorkspaceRoot(traversalPath) {
		t.Fatalf("expected directory traversal to be detected: %s", traversalPath)
	}
}
