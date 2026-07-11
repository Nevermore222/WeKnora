package executor

import (
	"testing"

	"github.com/Tencent/Xelora/internal/types"
)

func TestResolveConversationOutputContextUnboundWhenBindingNil(t *testing.T) {
	ctx := ResolveConversationOutputContext("sess-1", nil)
	if ctx.Mode != ConversationOutputModeUnbound {
		t.Fatalf("expected unbound mode, got %s", ctx.Mode)
	}
	if ctx.WriteAllowed {
		t.Fatalf("expected write not allowed for nil binding")
	}
}

func TestResolveConversationOutputContextUnboundWhenStatusUnbound(t *testing.T) {
	ctx := ResolveConversationOutputContext("sess-1", &types.SessionWorkspaceBinding{
		WorkspaceID: "tenant:1",
		Status:      types.SessionWorkspaceBindingStatusUnbound,
	})
	if ctx.Mode != ConversationOutputModeUnbound {
		t.Fatalf("expected unbound mode, got %s", ctx.Mode)
	}
}

func TestResolveConversationOutputContextBoundWhenStatusBound(t *testing.T) {
	ctx := ResolveConversationOutputContext("sess-1", &types.SessionWorkspaceBinding{
		WorkspaceID: "tenant:1",
		RootPath:    "/data/workspaces/tenant-1",
		Status:      types.SessionWorkspaceBindingStatusBound,
	})
	if ctx.Mode != ConversationOutputModeBound {
		t.Fatalf("expected bound mode, got %s", ctx.Mode)
	}
	if !ctx.WriteAllowed {
		t.Fatalf("expected write allowed for bound status")
	}
	if ctx.EffectiveRootDir != "/data/workspaces/tenant-1" {
		t.Fatalf("expected root path, got %s", ctx.EffectiveRootDir)
	}
}

func TestResolveConversationOutputContextBlockedWhenAccessDenied(t *testing.T) {
	ctx := ResolveConversationOutputContext("sess-1", &types.SessionWorkspaceBinding{
		WorkspaceID:       "tenant:1",
		Status:            types.SessionWorkspaceBindingStatusAccessDenied,
		ValidationMessage: "user lost access",
	})
	if ctx.Mode != ConversationOutputModeBound {
		t.Fatalf("expected bound mode (blocked), got %s", ctx.Mode)
	}
	if ctx.WriteAllowed {
		t.Fatalf("expected write not allowed for access_denied")
	}
	if ctx.FailureCode != ConversationOutputFailureAccessDenied {
		t.Fatalf("expected access_denied failure code, got %s", ctx.FailureCode)
	}
	if ctx.FailureMessage != "user lost access" {
		t.Fatalf("expected validation message preserved, got %s", ctx.FailureMessage)
	}
}

func TestIsWithinWorkspaceRootBlocksPathEscape(t *testing.T) {
	ctx := ConversationOutputContext{
		Mode:             ConversationOutputModeBound,
		EffectiveRootDir: "/data/workspaces/tenant-1",
		WriteAllowed:     true,
	}
	tests := []struct {
		path    string
		want    bool
	}{
		{"/data/workspaces/tenant-1/report.md", true},
		{"/data/workspaces/tenant-1", true},
		{"/data/workspaces/tenant-1/../tenant-2/secret.md", false},
		{"/etc/passwd", false},
		{"/data/workspaces/tenant-1-extra/file.md", false},
	}
	for _, tt := range tests {
		got := ctx.IsWithinWorkspaceRoot(tt.path)
		if got != tt.want {
			t.Errorf("IsWithinWorkspaceRoot(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIsWithinWorkspaceRootReturnsFalseForUnbound(t *testing.T) {
	ctx := ConversationOutputContext{Mode: ConversationOutputModeUnbound}
	if ctx.IsWithinWorkspaceRoot("/anywhere/file.md") {
		t.Fatalf("expected false for unbound context")
	}
}
