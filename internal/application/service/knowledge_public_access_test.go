package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

// stubPublicKBService resolves knowledge bases by ID so the document-level
// access path can inspect visibility.
type stubPublicKBService struct {
	interfaces.KnowledgeBaseService
	byID map[string]*types.KnowledgeBase
}

func (s *stubPublicKBService) GetKnowledgeBaseByID(
	_ context.Context, id string,
) (*types.KnowledgeBase, error) {
	if kb := s.byID[id]; kb != nil {
		return kb, nil
	}
	return nil, errors.New("knowledge base not found")
}

// A document inside a system-public knowledge base owned by another workspace
// must be resolvable for @-mention retrieval. Whole-KB search already honours
// visibility=public, so dropping the document here makes the file silently
// unreferenceable while its knowledge base stays visible and searchable.
func TestGetKnowledgeBatchWithSharedAccess_IncludesPublicKBKnowledge(t *testing.T) {
	service, db := newKnowledgeSharedAccessService(t, &fakeKBShareService{
		allowedKBs: map[string]bool{},
	})
	service.kbService = &stubPublicKBService{byID: map[string]*types.KnowledgeBase{
		"kb-public": {
			ID:         "kb-public",
			TenantID:   2,
			Visibility: types.ResourceVisibilityPublic,
		},
	}}
	seedKnowledge(t, db, &types.Knowledge{
		ID:              "k-public",
		TenantID:        2,
		KnowledgeBaseID: "kb-public",
		Type:            "file",
		Title:           "public doc",
		FileName:        "public.txt",
		FileType:        "txt",
		ParseStatus:     types.ParseStatusCompleted,
		EnableStatus:    "enabled",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	got, err := service.GetKnowledgeBatchWithSharedAccess(
		newSharedAccessContext(), 1, []string{"k-public"},
	)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "k-public", got[0].ID)
	require.Equal(t, uint64(2), got[0].TenantID)
}

// Regression guard: the public branch must not widen into a general
// cross-workspace document bypass. A workspace-visible KB with no organization
// share stays out of scope.
func TestGetKnowledgeBatchWithSharedAccess_ExcludesForeignWorkspaceKBKnowledge(t *testing.T) {
	service, db := newKnowledgeSharedAccessService(t, &fakeKBShareService{
		allowedKBs: map[string]bool{},
	})
	service.kbService = &stubPublicKBService{byID: map[string]*types.KnowledgeBase{
		"kb-workspace": {
			ID:         "kb-workspace",
			TenantID:   2,
			Visibility: types.ResourceVisibilityWorkspace,
		},
	}}
	seedKnowledge(t, db, &types.Knowledge{
		ID:              "k-workspace",
		TenantID:        2,
		KnowledgeBaseID: "kb-workspace",
		Type:            "file",
		Title:           "workspace doc",
		FileName:        "workspace.txt",
		FileType:        "txt",
		ParseStatus:     types.ParseStatusCompleted,
		EnableStatus:    "enabled",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	got, err := service.GetKnowledgeBatchWithSharedAccess(
		newSharedAccessContext(), 1, []string{"k-workspace"},
	)

	require.NoError(t, err)
	require.Empty(t, got)
}
