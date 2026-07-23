package enterprise

import "time"

// ConnectionStatus represents the state of an enterprise server connection.
type ConnectionStatus string

const (
	StatusConnected    ConnectionStatus = "connected"
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusConnecting   ConnectionStatus = "connecting"
	StatusError        ConnectionStatus = "error"
)

// ServerConfig stores the connection configuration for an enterprise Xelora server.
// Persisted in the local SQLite database.
type ServerConfig struct {
	ID          string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name        string     `json:"name" gorm:"type:varchar(255);not null"`
	BaseURL     string     `json:"base_url" gorm:"type:varchar(512);not null"`
	APIToken    string     `json:"-" gorm:"type:text"`
	AutoConnect bool       `json:"auto_connect" gorm:"default:true"`
	// Linked identity: the server-side user this client auto-provisioned itself
	// as. Sensitive fields (password/JWT/refresh) are json:"-" and encrypted at
	// rest; they are never returned to the frontend.
	LinkedUserID        string     `json:"linked_user_id,omitempty" gorm:"type:varchar(36)"`
	LinkedEmail         string     `json:"linked_email,omitempty" gorm:"type:varchar(255)"`
	LinkedTenantID      string     `json:"linked_tenant_id,omitempty" gorm:"type:varchar(36)"`
	LinkedPassword      string     `json:"-" gorm:"type:text"`
	ServerJWT           string     `json:"-" gorm:"type:text"`
	ServerRefreshToken  string     `json:"-" gorm:"type:text"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
}

func (ServerConfig) TableName() string {
	return "enterprise_servers"
}

// ServerCapabilities describes what an enterprise server exposes.
type ServerCapabilities struct {
	KnowledgeBases []RemoteKnowledgeBase `json:"knowledge_bases"`
	Agents         []RemoteAgent         `json:"agents"`
	Skills         []RemoteSkill         `json:"skills"`
	Models         []RemoteModel         `json:"models"`
}

// RemoteKnowledgeBase is a lightweight reference to an enterprise knowledge base.
type RemoteKnowledgeBase struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	DocCount    int    `json:"doc_count,omitempty"`
	// Shared marks KBs granted via an organization (fetched with the linked JWT
	// from /shared-knowledge-bases), as opposed to the server tenant's own KBs.
	Shared     bool   `json:"shared,omitempty"`
	Permission string `json:"permission,omitempty"`
	OrgName    string `json:"org_name,omitempty"`
}

// RemoteAgent is a lightweight reference to an enterprise agent.
type RemoteAgent struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	// Shared marks agents granted via an organization (fetched with the linked
	// JWT from /shared-agents).
	Shared     bool   `json:"shared,omitempty"`
	Permission string `json:"permission,omitempty"`
	OrgName    string `json:"org_name,omitempty"`
}

// RemoteSkill is a lightweight reference to an enterprise skill.
type RemoteSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// RemoteModel is a lightweight reference to a model available on the enterprise server.
type RemoteModel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider,omitempty"`
}

// ResourceCacheEntry stores cached enterprise resource metadata for instant UI
// rendering on reconnect. Not authoritative — refreshed on every successful connect.
type ResourceCacheEntry struct {
	ID           string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	ServerID     string    `json:"server_id" gorm:"type:varchar(36);index;not null"`
	ResourceType string    `json:"resource_type" gorm:"type:varchar(32);not null"`
	ResourceID   string    `json:"resource_id" gorm:"type:varchar(36);not null"`
	Metadata     string    `json:"metadata" gorm:"type:text"`
	CachedAt     time.Time `json:"cached_at"`
}

func (ResourceCacheEntry) TableName() string {
	return "enterprise_resource_cache"
}

// HealthEvent is emitted by the connector when a server's health status changes.
type HealthEvent struct {
	ServerID string
	Status   ConnectionStatus
	Error    string
	Time     time.Time
}

// AggregatedResource is a unified view of a resource that can come from either
// a local or enterprise source. Used by the frontend to render merged lists.
type AggregatedResource struct {
	ID           string `json:"id"`
	Type         string `json:"type"` // "knowledge_base", "agent", "skill", "model"
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Origin       string `json:"origin"` // "local" or "enterprise"
	ServerID     string `json:"server_id,omitempty"`
	ServerName   string `json:"server_name,omitempty"`
	Available    bool   `json:"available"`
	// Shared marks resources granted via an organization (as opposed to the
	// server tenant's own resources); Permission/OrgName describe the grant.
	Shared     bool   `json:"shared,omitempty"`
	Permission string `json:"permission,omitempty"`
	OrgName    string `json:"org_name,omitempty"`
}
