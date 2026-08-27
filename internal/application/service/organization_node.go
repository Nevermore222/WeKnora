package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/Xelora/internal/application/repository"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/google/uuid"
)

var (
	ErrOrganizationNodeInvalid   = errors.New("invalid organization node")
	ErrOrganizationNodeDepth     = errors.New("organization node depth is not supported")
	ErrOrganizationNodeName      = errors.New("organization node name is required")
	ErrOrganizationNodeDuplicate = errors.New("organization node name already exists")
)

type organizationNodeService struct {
	repo interfaces.OrganizationNodeRepository
}

func NewOrganizationNodeService(repo interfaces.OrganizationNodeRepository) interfaces.OrganizationNodeService {
	return &organizationNodeService{repo: repo}
}

func (s *organizationNodeService) ListTree(ctx context.Context, tenantID uint64) ([]*types.OrganizationNode, error) {
	nodes, err := s.repo.ListByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*types.OrganizationNode, len(nodes))
	roots := make([]*types.OrganizationNode, 0)
	for _, node := range nodes {
		node.Children = nil
		byID[node.ID] = node
	}
	for _, node := range nodes {
		if node.ParentID == nil || *node.ParentID == "" {
			roots = append(roots, node)
			continue
		}
		if parent := byID[*node.ParentID]; parent != nil {
			parent.Children = append(parent.Children, node)
		}
	}
	return roots, nil
}

func (s *organizationNodeService) ListResources(ctx context.Context, tenantID uint64, id string) (*types.OrganizationNodeResources, error) {
	node, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	ids, err := s.repo.ListDescendantIDs(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	kbs, err := s.repo.ListKnowledgeBases(ctx, tenantID, ids)
	if err != nil {
		return nil, err
	}
	agents, err := s.repo.ListAgents(ctx, tenantID, ids)
	if err != nil {
		return nil, err
	}
	unassignedKBs, err := s.repo.ListUnassignedKnowledgeBases(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	unassignedAgents, err := s.repo.ListUnassignedAgents(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return &types.OrganizationNodeResources{Node: node, KnowledgeBases: kbs, Agents: agents, UnassignedKnowledgeBases: unassignedKBs, UnassignedAgents: unassignedAgents}, nil
}

func (s *organizationNodeService) AssignResource(ctx context.Context, tenantID uint64, id string, req *types.AssignOrganizationNodeResourceRequest) error {
	id = strings.TrimSpace(id)
	if id == "" || req == nil || strings.TrimSpace(req.ResourceID) == "" {
		return ErrOrganizationNodeInvalid
	}
	if _, err := s.repo.GetByID(ctx, tenantID, id); err != nil {
		return err
	}
	nodeID := &id
	switch req.ResourceType {
	case "knowledge_base":
		return s.repo.AssignKnowledgeBase(ctx, tenantID, strings.TrimSpace(req.ResourceID), nodeID)
	case "agent":
		return s.repo.AssignAgent(ctx, tenantID, strings.TrimSpace(req.ResourceID), nodeID)
	default:
		return ErrOrganizationNodeInvalid
	}
}

func (s *organizationNodeService) Create(ctx context.Context, tenantID uint64, req *types.CreateOrganizationNodeRequest) (*types.OrganizationNode, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrOrganizationNodeName
	}
	if !req.Type.IsValid() {
		return nil, ErrOrganizationNodeInvalid
	}
	if req.Type == types.OrganizationNodeCompany && req.ParentID != nil {
		return nil, ErrOrganizationNodeInvalid
	}
	if req.Type != types.OrganizationNodeCompany && req.ParentID == nil {
		return nil, ErrOrganizationNodeInvalid
	}
	depth := 0
	if req.ParentID != nil {
		parent, err := s.repo.GetByID(ctx, tenantID, *req.ParentID)
		if err != nil {
			return nil, err
		}
		if (parent.Type == types.OrganizationNodeCompany && req.Type != types.OrganizationNodeDepartment) || (parent.Type == types.OrganizationNodeDepartment && req.Type != types.OrganizationNodeGroup) || parent.Type == types.OrganizationNodeGroup {
			return nil, ErrOrganizationNodeInvalid
		}
		depth = parent.Depth + 1
	}
	if depth > 2 {
		return nil, ErrOrganizationNodeDepth
	}
	if req.Type == types.OrganizationNodeCompany {
		count, err := s.repo.CountByType(ctx, tenantID, types.OrganizationNodeCompany)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, ErrOrganizationNodeInvalid
		}
	}
	exists, err := s.repo.NameExists(ctx, tenantID, req.ParentID, name, "")
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrOrganizationNodeDuplicate
	}
	now := time.Now()
	node := &types.OrganizationNode{ID: uuid.New().String(), TenantID: tenantID, ParentID: req.ParentID, Name: name, Type: req.Type, Depth: depth, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.Create(ctx, node); err != nil {
		return nil, err
	}
	return node, nil
}

func (s *organizationNodeService) UpdateName(ctx context.Context, tenantID uint64, id string, req *types.UpdateOrganizationNodeRequest) (*types.OrganizationNode, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrOrganizationNodeName
	}
	node, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	exists, err := s.repo.NameExists(ctx, tenantID, node.ParentID, name, id)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrOrganizationNodeDuplicate
	}
	if err := s.repo.UpdateName(ctx, tenantID, id, name); err != nil {
		return nil, err
	}
	node.Name = name
	return node, nil
}

func (s *organizationNodeService) Delete(ctx context.Context, tenantID uint64, id string) error {
	if _, err := s.repo.GetByID(ctx, tenantID, id); err != nil {
		return err
	}
	count, err := s.repo.CountChildren(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return repository.ErrOrganizationNodeHasChildren
	}
	return s.repo.Delete(ctx, tenantID, id)
}
