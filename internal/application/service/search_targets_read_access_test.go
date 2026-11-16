package service

import (
	"context"
	"testing"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

// fixtureKBRepo backs a real knowledgeBaseService so search-target tests exercise
// the production read rule instead of a re-implementation in a stub.
type fixtureKBRepo struct {
	interfaces.KnowledgeBaseRepository
	kbs []*types.KnowledgeBase
}

func (r *fixtureKBRepo) GetKnowledgeBaseByIDs(
	_ context.Context, ids []string,
) ([]*types.KnowledgeBase, error) {
	byID := make(map[string]*types.KnowledgeBase, len(r.kbs))
	for _, kb := range r.kbs {
		if kb != nil {
			byID[kb.ID] = kb
		}
	}
	out := make([]*types.KnowledgeBase, 0, len(ids))
	for _, id := range ids {
		if kb := byID[id]; kb != nil {
			out = append(out, kb)
		}
	}
	return out, nil
}

func newSearchTargetService(
	kbs []*types.KnowledgeBase,
	share interfaces.KBShareService,
) *sessionService {
	return &sessionService{
		knowledgeBaseService: &knowledgeBaseService{
			repo:           &fixtureKBRepo{kbs: kbs},
			kbShareService: share,
		},
		kbShareService: share,
	}
}

// An unreadable knowledge base must be dropped from the search scope. Substituting
// the retrieval tenant instead made the pipeline query another tenant's index for
// a KB that lives elsewhere: the answer came back empty with no diagnostic, and a
// single stale mention could fail the whole retrieval.
func TestBuildSearchTargetsForAccess_DropsUnreadableKB(t *testing.T) {
	svc := newSearchTargetService([]*types.KnowledgeBase{
		{ID: "kb-readable", TenantID: 10007, Visibility: types.ResourceVisibilityWorkspace},
		{ID: "kb-unreadable", TenantID: 10009, Visibility: types.ResourceVisibilityWorkspace},
	}, &fakeKBShareService{allowedKBs: map[string]bool{}})
	ctx := context.WithValue(context.Background(), types.UserIDContextKey, "wangl")

	got, err := svc.buildSearchTargetsForAccess(ctx, 10000, 10007, []string{"kb-readable", "kb-unreadable"}, nil)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "kb-readable", got[0].KnowledgeBaseID)
	require.Equal(t, uint64(10007), got[0].TenantID)
}

// A knowledge base shared with the caller's tenant stays searchable through a
// shared agent whose source tenant is different, and keeps its own tenant.
func TestBuildSearchTargetsForAccess_KeepsKBSharedWithCallerTenant(t *testing.T) {
	svc := newSearchTargetService([]*types.KnowledgeBase{
		{ID: "kb-shared", TenantID: 10009, Visibility: types.ResourceVisibilityWorkspace},
	}, &fakeKBShareService{allowedKBs: map[string]bool{"kb-shared": true}})
	ctx := context.WithValue(context.Background(), types.UserIDContextKey, "wangl")

	got, err := svc.buildSearchTargetsForAccess(ctx, 10000, 10007, []string{"kb-shared"}, nil)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, uint64(10009), got[0].TenantID)
}
