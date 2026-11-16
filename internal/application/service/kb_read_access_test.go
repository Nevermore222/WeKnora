package service

import (
	"context"
	"testing"

	apperrors "github.com/Tencent/Xelora/internal/errors"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// searchAuthContext builds a chat-path context: TenantIDContextKey carries the
// retrieval tenant (a shared agent's source tenant) while CallerTenantIDContextKey
// carries the authenticated caller's own tenant.
func searchAuthContext(retrievalTenantID, callerTenantID uint64) context.Context {
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, retrievalTenantID)
	ctx = context.WithValue(ctx, types.CallerTenantIDContextKey, callerTenantID)
	return ctx
}

// A knowledge base shared with the caller's tenant must stay searchable when the
// retrieval tenant is a shared agent's source tenant. Scope resolution and
// retrieval authorization previously ran against different tenants, so the KB
// was listed for the agent but retrieval answered "knowledge base not found".
func TestAuthorizeKBAccess_KBSharedWithCallerTenant(t *testing.T) {
	share := &fakeKBShareForAuth{
		allowed: map[string]map[uint64]bool{"kb-foreign": {10007: true}},
	}
	s := &knowledgeBaseService{kbShareService: share}
	kbs := []*types.KnowledgeBase{{
		ID:         "kb-foreign",
		TenantID:   99,
		Visibility: types.ResourceVisibilityWorkspace,
	}}

	err := s.authorizeKBAccess(searchAuthContext(10011, 10007), kbs, 10011)
	require.NoError(t, err)
}

// Regression guard: a foreign workspace knowledge base that is neither public,
// shared with the retrieval tenant nor shared with the caller stays rejected -
// and stays NotFound so its existence is not leaked.
func TestAuthorizeKBAccess_ForeignWorkspaceKBStillDenied(t *testing.T) {
	s := &knowledgeBaseService{kbShareService: &fakeKBShareForAuth{}}
	kbs := []*types.KnowledgeBase{{
		ID:         "kb-foreign",
		TenantID:   99,
		Visibility: types.ResourceVisibilityWorkspace,
	}}

	err := s.authorizeKBAccess(searchAuthContext(10011, 10007), kbs, 10011)
	require.Error(t, err)
	app, ok := apperrors.IsAppError(err)
	require.True(t, ok, "expected typed AppError, got %T", err)
	assert.Equal(t, apperrors.ErrNotFound, app.Code)
}

// Regression guard: a private knowledge base belonging to another user inside the
// retrieval tenant is not readable just because the tenant matches.
func TestAuthorizeKBAccess_PrivateKBOfAnotherUserDenied(t *testing.T) {
	s := &knowledgeBaseService{kbShareService: &fakeKBShareForAuth{}}
	ctx := context.WithValue(searchAuthContext(7, 7), types.UserIDContextKey, "caller")
	kbs := []*types.KnowledgeBase{{
		ID:         "kb-private",
		TenantID:   7,
		Visibility: types.ResourceVisibilityPrivate,
		CreatorID:  "someone-else",
	}}

	err := s.authorizeKBAccess(ctx, kbs, 7)
	require.Error(t, err)
	app, ok := apperrors.IsAppError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrNotFound, app.Code)
}
