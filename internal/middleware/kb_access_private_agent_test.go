package middleware

import (
	"context"
	"testing"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/stretchr/testify/require"
)

func privateAgentStub(mode string, kbIDs ...string) *stubAgentShareForGuard {
	return &stubAgentShareForGuard{
		agents: map[string]*types.CustomAgent{
			"agent-A": {
				ID:       "agent-A",
				TenantID: 200,
				Config: types.CustomAgentConfig{
					KBSelectionMode: mode,
					KnowledgeBases:  kbIDs,
				},
			},
		},
	}
}

func privateKBFixture() *types.KnowledgeBase {
	return &types.KnowledgeBase{
		ID:         "kb-private",
		TenantID:   200,
		Visibility: types.ResourceVisibilityPrivate,
		CreatorID:  "owner-200",
	}
}

// A private knowledge base stays creator-only even when a reachable agent lists
// it. Retrieval excludes private rows from every agent scope, so granting read
// here would surface a KB the UI opens but the agent can never search.
func TestRequireKBAccess_AgentShare_ModeSelected_DeniesPrivateKB(t *testing.T) {
	_, c := runGuard(t, 100, "kb-private",
		types.OrgRoleViewer,
		privateKBFixture(),
		nil,
		guardOpts{agentShare: privateAgentStub("selected", "kb-private"), agentID: "agent-A"},
	)
	require.True(t, c.IsAborted(), "private KB must not be readable through a shared agent")
}

// mode=all is broader still, so it must not become a way into another user's
// private rows inside the agent's workspace.
func TestRequireKBAccess_AgentShare_ModeAll_DeniesPrivateKB(t *testing.T) {
	_, c := runGuard(t, 100, "kb-private",
		types.OrgRoleViewer,
		privateKBFixture(),
		nil,
		guardOpts{agentShare: privateAgentStub("all"), agentID: "agent-A"},
	)
	require.True(t, c.IsAborted(), "private KB must not be readable through a mode=all agent")
}

// The creator keeps access to their own private KB, so the carve-out cannot lock
// owners out of their own rows.
func TestRequireKBAccess_AgentShare_AllowsOwnPrivateKB(t *testing.T) {
	_, c := runGuard(t, 200, "kb-private",
		types.OrgRoleViewer,
		privateKBFixture(),
		nil,
		guardOpts{agentShare: privateAgentStub("selected", "kb-private"), agentID: "agent-A", userID: "owner-200"},
	)
	require.False(t, c.IsAborted(), "the creator must keep read access to their own private KB")
}

func TestAgentMayCarryKB(t *testing.T) {
	creatorCtx := context.WithValue(context.Background(), types.UserIDContextKey, "owner-200")
	otherCtx := context.WithValue(context.Background(), types.UserIDContextKey, "someone-else")

	require.True(t, AgentMayCarryKB(otherCtx, &types.KnowledgeBase{
		ID: "kb-public", Visibility: types.ResourceVisibilityPublic,
	}))
	require.True(t, AgentMayCarryKB(otherCtx, &types.KnowledgeBase{
		ID: "kb-workspace", Visibility: types.ResourceVisibilityWorkspace,
	}))
	require.False(t, AgentMayCarryKB(otherCtx, privateKBFixture()))
	require.True(t, AgentMayCarryKB(creatorCtx, privateKBFixture()))
	require.False(t, AgentMayCarryKB(creatorCtx, nil))
}
