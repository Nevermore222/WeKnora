package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/Tencent/Xelora/internal/agent/tools"
	"github.com/Tencent/Xelora/internal/types"
)

func (s *Server) allowedKnowledgeBases(ctx context.Context, ep *types.MCPEndpoint) ([]*types.KnowledgeBase, error) {
	if !ep.RestrictsKnowledgeBases() {
		return s.kbService.ListVisibleKnowledgeBases(ctx)
	}
	out := make([]*types.KnowledgeBase, 0, len(ep.KnowledgeBaseIDs))
	for _, id := range ep.KnowledgeBaseIDs {
		kb, err := s.authorizedKnowledgeBase(ctx, id, types.OrgRoleViewer)
		if err == nil && kb != nil {
			out = append(out, kb)
		}
	}
	return out, nil
}

func (s *Server) authorizedKnowledgeBase(ctx context.Context, kbID string, _ types.OrgMemberRole) (*types.KnowledgeBase, error) {
	kb, err := s.kbService.GetKnowledgeBaseByIDOnly(ctx, strings.TrimSpace(kbID))
	if err != nil || kb == nil {
		return nil, fmt.Errorf("knowledge base not found")
	}
	if !s.endpointAllowsKnowledgeBase(ctx, kb) {
		return nil, fmt.Errorf("knowledge base is outside endpoint scope")
	}
	return kb, nil
}

func (s *Server) endpointAllowsKnowledgeBase(ctx context.Context, kb *types.KnowledgeBase) bool {
	callerTenant, _ := types.TenantIDFromContext(ctx)
	if kb.TenantID == callerTenant {
		return true
	}
	if kb.Visibility == types.ResourceVisibilityPublic {
		return true
	}
	if s.kbShareService != nil {
		role, shared, err := s.kbShareService.CheckTenantKBPermission(ctx, kb.ID, callerTenant, types.TenantRoleFromContext(ctx))
		return err == nil && shared && role.HasPermission(types.OrgRoleViewer)
	}
	return false
}

func (s *Server) resolveKB(ctx context.Context, kb *types.KnowledgeBase, required types.OrgMemberRole) error {
	if required != types.OrgRoleViewer {
		return fmt.Errorf("write operations are not enabled for this MCP endpoint")
	}
	if !s.endpointAllowsKnowledgeBase(ctx, kb) {
		return fmt.Errorf("knowledge base access denied")
	}
	return nil
}

func (s *Server) scopedKBContext(ctx context.Context, kb *types.KnowledgeBase, required types.OrgMemberRole) (context.Context, error) {
	if err := s.resolveKB(ctx, kb, required); err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, types.TenantIDContextKey, kb.TenantID), nil
}

func (s *Server) selectKnowledgeBases(ctx context.Context, ep *types.MCPEndpoint, requested []string) ([]*types.KnowledgeBase, error) {
	allowed, err := s.allowedKnowledgeBases(ctx, ep)
	if err != nil {
		return nil, err
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("no knowledge base is available on this endpoint")
	}
	if len(requested) == 0 {
		return allowed, nil
	}
	out := make([]*types.KnowledgeBase, 0, len(requested))
	seen := map[string]struct{}{}
	for _, selector := range requested {
		selector = strings.TrimSpace(selector)
		var found *types.KnowledgeBase
		for _, kb := range allowed {
			if kb.ID == selector || strings.EqualFold(kb.Name, selector) {
				found = kb
				break
			}
		}
		if found == nil {
			return nil, fmt.Errorf("knowledge base %q was not found or is outside this endpoint's scope", selector)
		}
		if _, ok := seen[found.ID]; !ok {
			out = append(out, found)
			seen[found.ID] = struct{}{}
		}
	}
	return out, nil
}

func searchTargetsFor(kbs []*types.KnowledgeBase) types.SearchTargets {
	targets := make(types.SearchTargets, 0, len(kbs))
	for _, kb := range kbs {
		targets = append(targets, &types.SearchTarget{Type: types.SearchTargetTypeKnowledgeBase, KnowledgeBaseID: kb.ID, TenantID: kb.TenantID})
	}
	return targets
}
func knowledgeBaseIDs(kbs []*types.KnowledgeBase) []string {
	out := make([]string, 0, len(kbs))
	for _, kb := range kbs {
		out = append(out, kb.ID)
	}
	return out
}
func retrievableKnowledgeBases(kbs []*types.KnowledgeBase) []*types.KnowledgeBase {
	out := make([]*types.KnowledgeBase, 0, len(kbs))
	for _, kb := range kbs {
		if kb.IsVectorEnabled() || kb.IsKeywordEnabled() {
			out = append(out, kb)
		}
	}
	return out
}
func wikiKnowledgeBases(kbs []*types.KnowledgeBase) []*types.KnowledgeBase {
	out := make([]*types.KnowledgeBase, 0, len(kbs))
	for _, kb := range kbs {
		if kb.IsWikiEnabled() {
			out = append(out, kb)
		}
	}
	return out
}
func wikiScopesFor(kbs []*types.KnowledgeBase) []tools.WikiScope {
	return tools.NewWikiScopesFromKBIDs(knowledgeBaseIDs(kbs))
}

func (s *Server) knowledgeInScope(ctx context.Context, ep *types.MCPEndpoint, knowledgeID string) (*types.Knowledge, *types.KnowledgeBase, error) {
	k, err := s.knowledgeService.GetKnowledgeByIDOnly(ctx, strings.TrimSpace(knowledgeID))
	if err != nil || k == nil {
		return nil, nil, fmt.Errorf("document %q was not found", knowledgeID)
	}
	allowed, err := s.allowedKnowledgeBases(ctx, ep)
	if err != nil {
		return nil, nil, err
	}
	for _, kb := range allowed {
		if kb.ID == k.KnowledgeBaseID {
			return k, kb, nil
		}
	}
	return nil, nil, fmt.Errorf("document %q is outside this endpoint's scope", knowledgeID)
}
