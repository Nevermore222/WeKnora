package interfaces

import (
	"context"
	"github.com/Tencent/Xelora/internal/types"
)

type ClassificationService interface {
	List(ctx context.Context, resourceType string) ([]*types.ClassificationModule, error)
	CreateModule(ctx context.Context, resourceType string, name string, sortOrder int) (*types.ClassificationModule, error)
	UpdateModule(ctx context.Context, resourceType string, id string, name *string, sortOrder *int, enabled *bool) (*types.ClassificationModule, error)
	DeleteModule(ctx context.Context, resourceType string, id string) error
	CreateOption(ctx context.Context, resourceType string, moduleID string, parentID *string, name string, sortOrder int) (*types.ClassificationOption, error)
	UpdateOption(ctx context.Context, resourceType string, id string, name *string, sortOrder *int, enabled *bool) (*types.ClassificationOption, error)
	DeleteOption(ctx context.Context, resourceType string, id string) error
	ValidateSelections(ctx context.Context, resourceType string, selections map[string][]string) error
}

type ClassificationRepository interface {
	ListModules(ctx context.Context, tenantID uint64, resourceType string) ([]*types.ClassificationModule, error)
	GetModule(ctx context.Context, tenantID uint64, id string, resourceType string) (*types.ClassificationModule, error)
	CreateModule(ctx context.Context, module *types.ClassificationModule) error
	UpdateModule(ctx context.Context, module *types.ClassificationModule) error
	DeleteModule(ctx context.Context, tenantID uint64, id string, resourceType string) error
	ListOptions(ctx context.Context, tenantID uint64, moduleID string, resourceType string) ([]*types.ClassificationOption, error)
	GetOption(ctx context.Context, tenantID uint64, id string) (*types.ClassificationOption, error)
	CreateOption(ctx context.Context, option *types.ClassificationOption) error
	UpdateOption(ctx context.Context, option *types.ClassificationOption) error
	DeleteOption(ctx context.Context, tenantID uint64, id string) error
}
