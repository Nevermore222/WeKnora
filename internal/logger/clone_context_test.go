package logger

import (
	"context"
	"testing"

	"github.com/Tencent/Xelora/internal/types"
)

// The chat pipeline runs on logger.CloneContext(...), so every authorization
// input that is not copied silently disappears before retrieval: a system admin
// degrades to an ordinary caller, the caller's own tenant collapses into the
// rewritten retrieval tenant, and agent-scoped knowledge base grants are lost.
// Each of those turns a knowledge base that is visible in the UI into one that
// cannot be searched.
func TestCloneContext_PreservesAuthorizationKeys(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, types.TenantIDContextKey, uint64(10000))
	ctx = context.WithValue(ctx, types.CallerTenantIDContextKey, uint64(10011))
	ctx = context.WithValue(ctx, types.SystemAdminContextKey, true)

	got := CloneContext(ctx)

	if v, ok := got.Value(types.CallerTenantIDContextKey).(uint64); !ok || v != 10011 {
		t.Fatalf("caller tenant = (%v, %v), want (10011, true)", v, ok)
	}
	if !types.IsSystemAdminFromContext(got) {
		t.Fatal("system admin flag must survive CloneContext")
	}
	if v, ok := types.TenantIDFromContext(got); !ok || v != 10000 {
		t.Fatalf("retrieval tenant = (%d, %v), want (10000, true)", v, ok)
	}
}
