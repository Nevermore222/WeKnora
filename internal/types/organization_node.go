package types

import "time"

// OrganizationNodeType identifies a node in the tenant's internal hierarchy.
type OrganizationNodeType string

const (
	OrganizationNodeCompany    OrganizationNodeType = "company"
	OrganizationNodeDepartment OrganizationNodeType = "department"
	OrganizationNodeGroup      OrganizationNodeType = "group"
)

func (t OrganizationNodeType) IsValid() bool {
	return t == OrganizationNodeCompany || t == OrganizationNodeDepartment || t == OrganizationNodeGroup
}

// OrganizationNode is the internal company/department/group tree. It is
// intentionally separate from Organization, which represents a cross-tenant
// collaboration space.
type OrganizationNode struct {
	ID        string               `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID  uint64               `json:"tenant_id" gorm:"not null;index"`
	ParentID  *string              `json:"parent_id,omitempty" gorm:"type:varchar(36);index"`
	Name      string               `json:"name" gorm:"type:varchar(255);not null"`
	Type      OrganizationNodeType `json:"type" gorm:"type:varchar(32);not null"`
	Depth     int                  `json:"depth" gorm:"not null"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`

	Children []*OrganizationNode `json:"children,omitempty" gorm:"-"`
}

func (OrganizationNode) TableName() string { return "organization_nodes" }

type CreateOrganizationNodeRequest struct {
	ParentID *string              `json:"parent_id"`
	Name     string               `json:"name" binding:"required"`
	Type     OrganizationNodeType `json:"type" binding:"required"`
}

type UpdateOrganizationNodeRequest struct {
	Name string `json:"name" binding:"required"`
}

type AssignOrganizationNodeResourceRequest struct {
	ResourceType string `json:"resource_type" binding:"required"`
	ResourceID   string `json:"resource_id" binding:"required"`
}

type OrganizationNodeResources struct {
	Node                     *OrganizationNode `json:"node"`
	KnowledgeBases           []*KnowledgeBase  `json:"knowledge_bases"`
	Agents                   []*CustomAgent    `json:"agents"`
	UnassignedKnowledgeBases []*KnowledgeBase  `json:"unassigned_knowledge_bases"`
	UnassignedAgents         []*CustomAgent    `json:"unassigned_agents"`
}
