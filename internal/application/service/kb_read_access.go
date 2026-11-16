package service

import (
	"context"

	"github.com/Tencent/Xelora/internal/types"
)

// CanReadKnowledgeBase reports whether the current caller may read kb. It is the
// single read-authorization rule shared by retrieval, search-target building and
// agent knowledge base scope resolution, so a knowledge base that is visible in
// the UI is always usable and one that is not visible is never searched under a
// substituted tenant.
//
// retrievalTenantID is the tenant the retrieval itself runs in: a shared or
// public agent's source tenant, which is not necessarily the caller's tenant.
func (s *knowledgeBaseService) CanReadKnowledgeBase(
	ctx context.Context,
	kb *types.KnowledgeBase,
	retrievalTenantID uint64,
) (bool, error) {
	if kb == nil {
		return false, nil
	}
	if types.IsSystemAdminFromContext(ctx) {
		return true, nil
	}

	scopeTenants := kbReadScopeTenantIDs(ctx, retrievalTenantID)
	callerUserID, _ := types.UserIDFromContext(ctx)
	for _, tenantID := range scopeTenants {
		if kb.TenantID != tenantID {
			continue
		}
		if kb.Visibility == types.ResourceVisibilityPrivate && kb.CreatorID != callerUserID {
			// Private rows stay creator-only. Fall through so an explicit public
			// or organization grant can still apply.
			continue
		}
		return true, nil
	}

	if kb.Visibility == types.ResourceVisibilityPublic {
		return true, nil
	}

	if s.kbShareService == nil {
		return false, nil
	}
	callerTenantRole := types.TenantRoleFromContext(ctx)
	for _, tenantID := range scopeTenants {
		allowed, err := s.kbShareService.HasTenantKBPermission(
			ctx, kb.ID, tenantID, callerTenantRole, types.OrgRoleViewer,
		)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

// kbReadScopeTenantIDs returns the distinct tenants whose entitlements count for
// this request: the retrieval tenant plus the authenticated caller's and the
// session owner's tenant.
//
// Retrieval scope and permission scope used to be evaluated against different
// tenants - a shared agent retrieves in its source tenant while the caller's
// shares are keyed on the caller's own tenant. Considering only one of them is
// what made an organization-shared knowledge base unsearchable through the very
// agent that listed it.
func kbReadScopeTenantIDs(ctx context.Context, retrievalTenantID uint64) []uint64 {
	ids := make([]uint64, 0, 3)
	seen := make(map[uint64]bool, 3)
	add := func(v uint64) {
		if v == 0 || seen[v] {
			return
		}
		seen[v] = true
		ids = append(ids, v)
	}
	// Caller-first: the caller's own entitlements are the specific grant, the
	// retrieval tenant is the resource owner's scope. Checking the caller first
	// also keeps audit logs pointed at the tenant the share was made to.
	if callerTenantID, ok := types.CallerTenantIDFromContext(ctx); ok {
		add(callerTenantID)
	}
	if sessionTenantID, ok := types.SessionTenantIDFromContext(ctx); ok {
		add(sessionTenantID)
	}
	add(retrievalTenantID)
	return ids
}
