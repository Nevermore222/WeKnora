package interfaces

import (
	"context"
	"github.com/Tencent/Xelora/internal/types"
)

type OrganizationNodeService interface {
	ListTree(ctx context.Context, tenantID uint64) ([]*types.OrganizationNode, error)
	ListResources(ctx context.Context, tenantID uint64, id string) (*types.OrganizationNodeResources, error)
	Create(ctx context.Context, tenantID uint64, req *types.CreateOrganizationNodeRequest) (*types.OrganizationNode, error)
	UpdateName(ctx context.Context, tenantID uint64, id string, req *types.UpdateOrganizationNodeRequest) (*types.OrganizationNode, error)
	AssignResource(ctx context.Context, tenantID uint64, id string, req *types.AssignOrganizationNodeResourceRequest) error
	Delete(ctx context.Context, tenantID uint64, id string) error
}

type OrganizationNodeRepository interface {
	ListByTenantID(ctx context.Context, tenantID uint64) ([]*types.OrganizationNode, error)
	GetByID(ctx context.Context, tenantID uint64, id string) (*types.OrganizationNode, error)
	Create(ctx context.Context, node *types.OrganizationNode) error
	UpdateName(ctx context.Context, tenantID uint64, id string, name string) error
	Delete(ctx context.Context, tenantID uint64, id string) error
	CountChildren(ctx context.Context, tenantID uint64, id string) (int64, error)
	CountByType(ctx context.Context, tenantID uint64, nodeType types.OrganizationNodeType) (int64, error)
	NameExists(ctx context.Context, tenantID uint64, parentID *string, name string, excludeID string) (bool, error)
	ListDescendantIDs(ctx context.Context, tenantID uint64, id string) ([]string, error)
	ListKnowledgeBases(ctx context.Context, tenantID uint64, nodeIDs []string) ([]*types.KnowledgeBase, error)
	ListAgents(ctx context.Context, tenantID uint64, nodeIDs []string) ([]*types.CustomAgent, error)
	ListUnassignedKnowledgeBases(ctx context.Context, tenantID uint64) ([]*types.KnowledgeBase, error)
	ListUnassignedAgents(ctx context.Context, tenantID uint64) ([]*types.CustomAgent, error)
	AssignKnowledgeBase(ctx context.Context, tenantID uint64, resourceID string, nodeID *string) error
	AssignAgent(ctx context.Context, tenantID uint64, resourceID string, nodeID *string) error
}
