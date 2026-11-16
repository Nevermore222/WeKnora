package service

import (
	"context"
	"testing"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type scopeVisibleKBStub struct {
	interfaces.KnowledgeBaseService
	visible []*types.KnowledgeBase
}

func (s *scopeVisibleKBStub) ListVisibleKnowledgeBases(context.Context) ([]*types.KnowledgeBase, error) {
	return s.visible, nil
}

type scopeSharedKBStub struct {
	fakeKBShareService
	shared []*types.SharedKnowledgeBaseInfo
}

func (s *scopeSharedKBStub) ListSharedKnowledgeBases(
	context.Context, uint64, types.TenantRole,
) ([]*types.SharedKnowledgeBaseInfo, error) {
	return s.shared, nil
}

// Global document search must cover every knowledge base the caller can see.
// The scope used to be built from the own tenant plus organization shares only,
// so documents inside system-public knowledge bases were retrievable in chat but
// could not be found by search: visible and openable, yet not referenceable.
func TestBuildKnowledgeSearchScopes_IncludesPublicAndSharedKBs(t *testing.T) {
	publicKB := &types.KnowledgeBase{
		ID:         "kb-public",
		TenantID:   10000,
		Type:       types.KnowledgeBaseTypeDocument,
		Visibility: types.ResourceVisibilityPublic,
	}
	ownKB := &types.KnowledgeBase{
		ID:       "kb-own",
		TenantID: 10007,
		Type:     types.KnowledgeBaseTypeDocument,
	}
	faqKB := &types.KnowledgeBase{
		ID:       "kb-faq",
		TenantID: 10000,
		Type:     types.KnowledgeBaseTypeFAQ,
	}
	sharedKB := &types.KnowledgeBase{
		ID:       "kb-shared",
		TenantID: 10009,
		Type:     types.KnowledgeBaseTypeDocument,
	}

	svc := &knowledgeService{
		kbService: &scopeVisibleKBStub{visible: []*types.KnowledgeBase{publicKB, ownKB, faqKB}},
		kbShareService: &scopeSharedKBStub{shared: []*types.SharedKnowledgeBaseInfo{{
			KnowledgeBase:  sharedKB,
			SourceTenantID: 10009,
			Permission:     types.OrgRoleViewer,
		}, {
			KnowledgeBase:  publicKB,
			SourceTenantID: 10000,
			Permission:     types.OrgRoleViewer,
		}}},
	}
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(10007))
	ctx = context.WithValue(ctx, types.UserIDContextKey, "wangl")

	got := svc.buildKnowledgeSearchScopes(ctx, 10007)

	byID := make(map[string]uint64, len(got))
	for _, scope := range got {
		byID[scope.KBID] = scope.TenantID
	}
	require.Len(t, got, 3, "expected own plus public plus shared document KBs, got %v", got)
	require.Equal(t, uint64(10007), byID["kb-own"])
	require.Equal(t, uint64(10000), byID["kb-public"], "public KB must be searched in its own tenant")
	require.Equal(t, uint64(10009), byID["kb-shared"], "shared KB must be searched in its source tenant")
	require.NotContains(t, byID, "kb-faq", "FAQ knowledge bases are not document search scopes")
}
