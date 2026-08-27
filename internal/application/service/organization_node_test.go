package service

import (
	"context"
	"strings"
	"testing"

	"github.com/Tencent/Xelora/internal/application/repository"
	"github.com/Tencent/Xelora/internal/types"
)

type organizationNodeRepoStub struct {
	nodes            map[string]*types.OrganizationNode
	assignedType     string
	assignedResource string
	assignedNodeID   *string
}

func (r *organizationNodeRepoStub) ListByTenantID(context.Context, uint64) ([]*types.OrganizationNode, error) {
	out := make([]*types.OrganizationNode, 0, len(r.nodes))
	for _, node := range r.nodes {
		copy := *node
		out = append(out, &copy)
	}
	return out, nil
}
func (r *organizationNodeRepoStub) GetByID(_ context.Context, tenantID uint64, id string) (*types.OrganizationNode, error) {
	node, ok := r.nodes[id]
	if !ok || node.TenantID != tenantID {
		return nil, repository.ErrOrganizationNodeNotFound
	}
	copy := *node
	return &copy, nil
}
func (r *organizationNodeRepoStub) Create(_ context.Context, node *types.OrganizationNode) error {
	copy := *node
	r.nodes[node.ID] = &copy
	return nil
}
func (r *organizationNodeRepoStub) UpdateName(_ context.Context, tenantID uint64, id, name string) error {
	node, ok := r.nodes[id]
	if !ok || node.TenantID != tenantID {
		return repository.ErrOrganizationNodeNotFound
	}
	node.Name = name
	return nil
}
func (r *organizationNodeRepoStub) Delete(_ context.Context, tenantID uint64, id string) error {
	node, ok := r.nodes[id]
	if !ok || node.TenantID != tenantID {
		return repository.ErrOrganizationNodeNotFound
	}
	delete(r.nodes, id)
	return nil
}
func (r *organizationNodeRepoStub) CountChildren(_ context.Context, tenantID uint64, id string) (int64, error) {
	var n int64
	for _, node := range r.nodes {
		if node.TenantID == tenantID && node.ParentID != nil && *node.ParentID == id {
			n++
		}
	}
	return n, nil
}
func (r *organizationNodeRepoStub) CountByType(_ context.Context, tenantID uint64, nodeType types.OrganizationNodeType) (int64, error) {
	var n int64
	for _, node := range r.nodes {
		if node.TenantID == tenantID && node.Type == nodeType {
			n++
		}
	}
	return n, nil
}
func (r *organizationNodeRepoStub) NameExists(_ context.Context, tenantID uint64, parentID *string, name, excludeID string) (bool, error) {
	for _, node := range r.nodes {
		if node.TenantID != tenantID || node.ID == excludeID || !strings.EqualFold(node.Name, name) {
			continue
		}
		if (node.ParentID == nil) == (parentID == nil) && (node.ParentID == nil || *node.ParentID == *parentID) {
			return true, nil
		}
	}
	return false, nil
}
func (r *organizationNodeRepoStub) ListDescendantIDs(_ context.Context, _ uint64, id string) ([]string, error) {
	return []string{id}, nil
}
func (r *organizationNodeRepoStub) ListKnowledgeBases(context.Context, uint64, []string) ([]*types.KnowledgeBase, error) {
	return nil, nil
}
func (r *organizationNodeRepoStub) ListAgents(context.Context, uint64, []string) ([]*types.CustomAgent, error) {
	return nil, nil
}
func (r *organizationNodeRepoStub) ListUnassignedKnowledgeBases(context.Context, uint64) ([]*types.KnowledgeBase, error) {
	return nil, nil
}
func (r *organizationNodeRepoStub) ListUnassignedAgents(context.Context, uint64) ([]*types.CustomAgent, error) {
	return nil, nil
}
func (r *organizationNodeRepoStub) AssignKnowledgeBase(_ context.Context, _ uint64, resourceID string, nodeID *string) error {
	r.assignedType, r.assignedResource, r.assignedNodeID = "knowledge_base", resourceID, nodeID
	return nil
}
func (r *organizationNodeRepoStub) AssignAgent(_ context.Context, _ uint64, resourceID string, nodeID *string) error {
	r.assignedType, r.assignedResource, r.assignedNodeID = "agent", resourceID, nodeID
	return nil
}

func TestOrganizationNodeServiceCreateEnforcesHierarchy(t *testing.T) {
	repo := &organizationNodeRepoStub{nodes: map[string]*types.OrganizationNode{}}
	svc := NewOrganizationNodeService(repo)
	ctx := context.Background()

	company, err := svc.Create(ctx, 1, &types.CreateOrganizationNodeRequest{Name: "Company", Type: types.OrganizationNodeCompany})
	if err != nil {
		t.Fatalf("create company: %v", err)
	}
	department, err := svc.Create(ctx, 1, &types.CreateOrganizationNodeRequest{Name: "Engineering", Type: types.OrganizationNodeDepartment, ParentID: &company.ID})
	if err != nil {
		t.Fatalf("create department: %v", err)
	}
	if _, err := svc.Create(ctx, 1, &types.CreateOrganizationNodeRequest{Name: "Invalid", Type: types.OrganizationNodeGroup, ParentID: &company.ID}); err != ErrOrganizationNodeInvalid {
		t.Fatalf("company child group error = %v", err)
	}
	if _, err := svc.Create(ctx, 1, &types.CreateOrganizationNodeRequest{Name: "Engineering", Type: types.OrganizationNodeDepartment, ParentID: &company.ID}); err != ErrOrganizationNodeDuplicate {
		t.Fatalf("duplicate department error = %v", err)
	}
	if _, err := svc.Create(ctx, 1, &types.CreateOrganizationNodeRequest{Name: "AI", Type: types.OrganizationNodeGroup, ParentID: &department.ID}); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := svc.Create(ctx, 1, &types.CreateOrganizationNodeRequest{Name: "Second", Type: types.OrganizationNodeCompany}); err != ErrOrganizationNodeInvalid {
		t.Fatalf("second company error = %v", err)
	}
}

func TestOrganizationNodeServiceAssignResourceUsesPathNode(t *testing.T) {
	repo := &organizationNodeRepoStub{nodes: map[string]*types.OrganizationNode{
		"target": {ID: "target", TenantID: 1, Name: "AI", Type: types.OrganizationNodeGroup, Depth: 2},
	}}
	svc := NewOrganizationNodeService(repo)

	if err := svc.AssignResource(context.Background(), 1, "target", &types.AssignOrganizationNodeResourceRequest{ResourceType: "knowledge_base", ResourceID: " kb-1 "}); err != nil {
		t.Fatalf("assign resource: %v", err)
	}
	if repo.assignedType != "knowledge_base" || repo.assignedResource != "kb-1" || repo.assignedNodeID == nil || *repo.assignedNodeID != "target" {
		t.Fatalf("assignment = type %q resource %q node %#v", repo.assignedType, repo.assignedResource, repo.assignedNodeID)
	}
	if err := svc.AssignResource(context.Background(), 2, "target", &types.AssignOrganizationNodeResourceRequest{ResourceType: "agent", ResourceID: "agent-1"}); err != repository.ErrOrganizationNodeNotFound {
		t.Fatalf("cross-tenant assignment error = %v", err)
	}
	if err := svc.AssignResource(context.Background(), 1, "target", &types.AssignOrganizationNodeResourceRequest{ResourceType: "agent"}); err != ErrOrganizationNodeInvalid {
		t.Fatalf("empty resource error = %v", err)
	}
}

func TestOrganizationNodeServiceListTreeAndLeafDelete(t *testing.T) {
	repo := &organizationNodeRepoStub{nodes: map[string]*types.OrganizationNode{}}
	svc := NewOrganizationNodeService(repo)
	companyID := "company"
	groupID := "group"
	repo.nodes[companyID] = &types.OrganizationNode{ID: companyID, TenantID: 1, Name: "Company", Type: types.OrganizationNodeCompany, Depth: 0}
	repo.nodes[groupID] = &types.OrganizationNode{ID: groupID, TenantID: 1, ParentID: &companyID, Name: "Group", Type: types.OrganizationNodeDepartment, Depth: 1}

	roots, err := svc.ListTree(context.Background(), 1)
	if err != nil || len(roots) != 1 || len(roots[0].Children) != 1 {
		t.Fatalf("tree = %#v, err=%v", roots, err)
	}
	if err := svc.Delete(context.Background(), 1, companyID); err != repository.ErrOrganizationNodeHasChildren {
		t.Fatalf("delete parent error = %v", err)
	}
	if err := svc.Delete(context.Background(), 1, groupID); err != nil {
		t.Fatalf("delete leaf: %v", err)
	}
}
