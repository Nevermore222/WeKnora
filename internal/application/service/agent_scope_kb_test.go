package service

import (
	"testing"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/stretchr/testify/require"
)

// agentScopeAllowsKB is the single rule behind "which knowledge bases may a
// caller retrieve through an agent they can use". These cases pin it down so the
// agent scope resolution, the KB detail page and retrieval cannot drift apart
// again: a knowledge base the UI opens must also be searchable.
func TestAgentScopeAllowsKB(t *testing.T) {
	agent := &types.CustomAgent{ID: "agent-1", TenantID: 10000}

	cases := []struct {
		name             string
		kb               *types.KnowledgeBase
		sessionTenantID  uint64
		sessionUserID    string
		sharedWithCaller bool
		want             bool
	}{
		{
			name:            "agent own workspace KB is readable through the agent",
			kb:              &types.KnowledgeBase{ID: "kb-1", TenantID: 10000, Visibility: types.ResourceVisibilityWorkspace},
			sessionTenantID: 10007,
			sessionUserID:   "wangl",
			want:            true,
		},
		{
			name:            "private KB of another user stays hidden",
			kb:              &types.KnowledgeBase{ID: "kb-2", TenantID: 10000, Visibility: types.ResourceVisibilityPrivate, CreatorID: "someone-else"},
			sessionTenantID: 10007,
			sessionUserID:   "wangl",
			want:            false,
		},
		{
			name:            "caller's own private KB in the caller's tenant stays readable",
			kb:              &types.KnowledgeBase{ID: "kb-3", TenantID: 10007, Visibility: types.ResourceVisibilityPrivate, CreatorID: "wangl"},
			sessionTenantID: 10007,
			sessionUserID:   "wangl",
			want:            true,
		},
		{
			name:            "third-tenant KB without a share is rejected",
			kb:              &types.KnowledgeBase{ID: "kb-4", TenantID: 10009, Visibility: types.ResourceVisibilityWorkspace},
			sessionTenantID: 10007,
			sessionUserID:   "wangl",
			want:            false,
		},
		{
			name:             "third-tenant KB shared with the caller is allowed",
			kb:               &types.KnowledgeBase{ID: "kb-5", TenantID: 10009, Visibility: types.ResourceVisibilityWorkspace},
			sessionTenantID:  10007,
			sessionUserID:    "wangl",
			sharedWithCaller: true,
			want:             true,
		},
		{
			name:            "public KB from any tenant is allowed",
			kb:              &types.KnowledgeBase{ID: "kb-6", TenantID: 10009, Visibility: types.ResourceVisibilityPublic},
			sessionTenantID: 10007,
			sessionUserID:   "wangl",
			want:            true,
		},
		{
			name:            "caller's own workspace KB is allowed",
			kb:              &types.KnowledgeBase{ID: "kb-7", TenantID: 10007, Visibility: types.ResourceVisibilityWorkspace},
			sessionTenantID: 10007,
			sessionUserID:   "wangl",
			want:            true,
		},
		{
			name:            "another user's private KB inside the caller's tenant is rejected",
			kb:              &types.KnowledgeBase{ID: "kb-8", TenantID: 10007, Visibility: types.ResourceVisibilityPrivate, CreatorID: "someone-else"},
			sessionTenantID: 10007,
			sessionUserID:   "wangl",
			want:            false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := agentScopeAllowsKB(agent, tc.kb, tc.sessionTenantID, tc.sessionUserID, tc.sharedWithCaller)
			require.Equal(t, tc.want, got)
		})
	}
}

// The agent-implied grant must reach retrieval as well: a configured KB owned by
// the agent's workspace is what the KB detail page already opens through the
// shared agent, so the chat pipeline has to search it instead of dropping it.
func TestFilterSharedAgentKnowledgeBases_AllowsAgentOwnWorkspaceKB(t *testing.T) {
	agent := &types.CustomAgent{TenantID: 10000}
	configured := []string{"kb-agent-workspace"}
	loaded := []*types.KnowledgeBase{{
		ID:         "kb-agent-workspace",
		TenantID:   10000,
		Visibility: types.ResourceVisibilityWorkspace,
	}}

	got := filterSharedAgentKnowledgeBases(agent, 10007, "wangl", configured, loaded, nil)

	require.Equal(t, configured, got)
}

// Private knowledge bases stay creator-only even when an agent references them,
// so publishing or sharing an agent cannot expose another user's private rows.
func TestFilterSharedAgentKnowledgeBases_RejectsAgentOwnPrivateKB(t *testing.T) {
	agent := &types.CustomAgent{TenantID: 10000}
	configured := []string{"kb-agent-private"}
	loaded := []*types.KnowledgeBase{{
		ID:         "kb-agent-private",
		TenantID:   10000,
		Visibility: types.ResourceVisibilityPrivate,
		CreatorID:  "someone-else",
	}}

	got := filterSharedAgentKnowledgeBases(agent, 10007, "wangl", configured, loaded, nil)

	require.Empty(t, got)
}
