package service

import (
	"github.com/Tencent/Xelora/internal/types"
)

// agentScopeAllowsKB decides which knowledge bases a caller may retrieve through
// an agent they are allowed to use. It is the single rule behind agent KB scope,
// so the knowledge bases a shared or public agent exposes in the UI are the same
// ones the chat pipeline actually searches - previously the UI granted read
// through the agent while retrieval dropped the same KB, which surfaced as "the
// agent answers but finds nothing in its knowledge base".
//
//   - private rows stay creator-only, wherever they live;
//   - the caller's own workspace rows are readable;
//   - system-public rows are readable from any workspace;
//   - rows explicitly shared with the caller's tenant are readable;
//   - rows owned by the agent's own workspace are readable, because publishing or
//     sharing that agent is a deliberate act by the owning workspace and the KB
//     detail page already grants the same read-only access through the agent;
//   - rows from a third workspace are never carried by the agent, so an agent
//     owner cannot hand out another workspace's resources.
func agentScopeAllowsKB(
	agent *types.CustomAgent,
	kb *types.KnowledgeBase,
	sessionTenantID uint64,
	sessionUserID string,
	sharedWithCaller bool,
) bool {
	if kb == nil {
		return false
	}
	if kb.Visibility == types.ResourceVisibilityPrivate &&
		!(sessionUserID != "" && kb.CreatorID == sessionUserID) {
		return false
	}
	if sessionTenantID != 0 && kb.TenantID == sessionTenantID {
		return true
	}
	if kb.Visibility == types.ResourceVisibilityPublic {
		return true
	}
	if sharedWithCaller {
		return true
	}
	return agent != nil && kb.TenantID == agent.TenantID
}
