package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// ClassificationTags stores selected leaf option IDs by classification module.
type ClassificationTags map[string][]string

func (t ClassificationTags) Value() (driver.Value, error) {
	if t == nil {
		// custom_agents.classification_tags is NOT NULL. Persist an empty
		// object for agents that have no selected classification tags instead
		// of returning SQL NULL.
		return []byte("{}"), nil
	}
	return json.Marshal(t)
}

func (t *ClassificationTags) Scan(src any) error {
	if src == nil {
		*t = ClassificationTags{}
		return nil
	}
	var raw []byte
	switch value := src.(type) {
	case []byte:
		raw = value
	case string:
		raw = []byte(value)
	default:
		return errors.New("ClassificationTags.Scan: unsupported source type")
	}
	if len(raw) == 0 {
		*t = ClassificationTags{}
		return nil
	}
	if err := json.Unmarshal(raw, t); err != nil {
		return err
	}
	if *t == nil {
		*t = ClassificationTags{}
	}
	return nil
}

// ClassificationModule is a system-managed dimension used to classify public resources.
// TenantID is kept for schema compatibility; system-wide modules use tenant_id = 0.
type ClassificationModule struct {
	ID           string                 `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID     uint64                 `json:"tenant_id" gorm:"not null;index"`
	ResourceType string                 `json:"resource_type" gorm:"type:varchar(50);not null;default:'knowledge_base';index:idx_classification_modules_resource_type"`
	Name         string                 `json:"name" gorm:"type:varchar(100);not null"`
	SortOrder    int                    `json:"sort_order" gorm:"not null;default:0"`
	Enabled      bool                   `json:"enabled" gorm:"not null;default:true"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Options      []ClassificationOption `json:"options,omitempty" gorm:"foreignKey:ModuleID"`
}

// ClassificationOption is a hierarchical option. ParentID is nil for a top-level option.
type ClassificationOption struct {
	ID           string                 `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID     uint64                 `json:"tenant_id" gorm:"not null;index"`
	ModuleID     string                 `json:"module_id" gorm:"type:varchar(36);not null;index"`
	ResourceType string                 `json:"resource_type" gorm:"type:varchar(50);not null;default:'knowledge_base';index:idx_classification_options_resource_type"`
	ParentID     *string                `json:"parent_id,omitempty" gorm:"type:varchar(36);index"`
	Name         string                 `json:"name" gorm:"type:varchar(100);not null"`
	SortOrder    int                    `json:"sort_order" gorm:"not null;default:0"`
	Enabled      bool                   `json:"enabled" gorm:"not null;default:true"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Children     []ClassificationOption `json:"children,omitempty" gorm:"-"`
	Path         []string               `json:"path,omitempty" gorm:"-"`
}

type KnowledgeBaseClassification struct {
	ModuleID   string   `json:"module_id"`
	ModuleName string   `json:"module_name"`
	OptionIDs  []string `json:"option_ids"`
	Options    []string `json:"options,omitempty"`
}
