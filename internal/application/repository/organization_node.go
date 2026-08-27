package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"gorm.io/gorm"
)

var (
	ErrOrganizationNodeNotFound     = errors.New("organization node not found")
	ErrOrganizationNodeHasChildren  = errors.New("organization node has children")
	ErrOrganizationResourceNotFound = errors.New("organization resource not found")
)

type organizationNodeRepository struct{ db *gorm.DB }

func NewOrganizationNodeRepository(db *gorm.DB) interfaces.OrganizationNodeRepository {
	return &organizationNodeRepository{db: db}
}

func (r *organizationNodeRepository) ListByTenantID(ctx context.Context, tenantID uint64) ([]*types.OrganizationNode, error) {
	var nodes []*types.OrganizationNode
	if err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("depth ASC, name ASC").Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

func (r *organizationNodeRepository) GetByID(ctx context.Context, tenantID uint64, id string) (*types.OrganizationNode, error) {
	var node types.OrganizationNode
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(id)).First(&node).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrOrganizationNodeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *organizationNodeRepository) Create(ctx context.Context, node *types.OrganizationNode) error {
	return r.db.WithContext(ctx).Create(node).Error
}

func (r *organizationNodeRepository) UpdateName(ctx context.Context, tenantID uint64, id string, name string) error {
	result := r.db.WithContext(ctx).Model(&types.OrganizationNode{}).Where("tenant_id = ? AND id = ?", tenantID, id).Update("name", name)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOrganizationNodeNotFound
	}
	return nil
}

func (r *organizationNodeRepository) Delete(ctx context.Context, tenantID uint64, id string) error {
	result := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&types.OrganizationNode{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOrganizationNodeNotFound
	}
	return nil
}

func (r *organizationNodeRepository) CountChildren(ctx context.Context, tenantID uint64, id string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&types.OrganizationNode{}).Where("tenant_id = ? AND parent_id = ?", tenantID, id).Count(&count).Error
	return count, err
}

func (r *organizationNodeRepository) CountByType(ctx context.Context, tenantID uint64, nodeType types.OrganizationNodeType) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&types.OrganizationNode{}).Where("tenant_id = ? AND type = ?", tenantID, nodeType).Count(&count).Error
	return count, err
}

func (r *organizationNodeRepository) NameExists(ctx context.Context, tenantID uint64, parentID *string, name string, excludeID string) (bool, error) {
	query := r.db.WithContext(ctx).Model(&types.OrganizationNode{}).
		Where("tenant_id = ? AND lower(name) = lower(?)", tenantID, strings.TrimSpace(name))
	if parentID == nil || strings.TrimSpace(*parentID) == "" {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", strings.TrimSpace(*parentID))
	}
	if excludeID != "" {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *organizationNodeRepository) ListDescendantIDs(ctx context.Context, tenantID uint64, id string) ([]string, error) {
	nodes, err := r.ListByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	children := make(map[string][]string)
	for _, node := range nodes {
		if node.ParentID != nil {
			children[*node.ParentID] = append(children[*node.ParentID], node.ID)
		}
	}
	ids := []string{id}
	for i := 0; i < len(ids); i++ {
		ids = append(ids, children[ids[i]]...)
	}
	return ids, nil
}

func (r *organizationNodeRepository) ListKnowledgeBases(ctx context.Context, tenantID uint64, nodeIDs []string) ([]*types.KnowledgeBase, error) {
	var items []*types.KnowledgeBase
	if len(nodeIDs) == 0 {
		return items, nil
	}
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND organization_node_id IN ?", tenantID, nodeIDs).Order("name ASC").Find(&items).Error
	return items, err
}

func (r *organizationNodeRepository) ListAgents(ctx context.Context, tenantID uint64, nodeIDs []string) ([]*types.CustomAgent, error) {
	var items []*types.CustomAgent
	if len(nodeIDs) == 0 {
		return items, nil
	}
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND organization_node_id IN ?", tenantID, nodeIDs).Order("name ASC").Find(&items).Error
	return items, err
}

func (r *organizationNodeRepository) ListUnassignedKnowledgeBases(ctx context.Context, tenantID uint64) ([]*types.KnowledgeBase, error) {
	var items []*types.KnowledgeBase
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND organization_node_id IS NULL", tenantID).Order("name ASC").Find(&items).Error
	return items, err
}

func (r *organizationNodeRepository) ListUnassignedAgents(ctx context.Context, tenantID uint64) ([]*types.CustomAgent, error) {
	var items []*types.CustomAgent
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND organization_node_id IS NULL AND is_builtin = ?", tenantID, false).Order("name ASC").Find(&items).Error
	return items, err
}

func (r *organizationNodeRepository) AssignKnowledgeBase(ctx context.Context, tenantID uint64, resourceID string, nodeID *string) error {
	result := r.db.WithContext(ctx).Model(&types.KnowledgeBase{}).Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(resourceID)).Update("organization_node_id", nodeID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOrganizationResourceNotFound
	}
	return nil
}

func (r *organizationNodeRepository) AssignAgent(ctx context.Context, tenantID uint64, resourceID string, nodeID *string) error {
	result := r.db.WithContext(ctx).Model(&types.CustomAgent{}).Where("tenant_id = ? AND id = ? AND is_builtin = ?", tenantID, strings.TrimSpace(resourceID), false).Update("organization_node_id", nodeID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOrganizationResourceNotFound
	}
	return nil
}
