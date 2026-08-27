package service

import (
	"context"
	"errors"
	apperrors "github.com/Tencent/Xelora/internal/errors"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/google/uuid"
	"strings"
)

type classificationService struct {
	repo interfaces.ClassificationRepository
}

func NewClassificationService(repo interfaces.ClassificationRepository) interfaces.ClassificationService {
	return &classificationService{repo: repo}
}
func tenant(ctx context.Context) (uint64, error) {
	id, ok := types.TenantIDFromContext(ctx)
	if !ok || id == 0 {
		return 0, errors.New("tenant context is required")
	}
	return id, nil
}
func (s *classificationService) List(ctx context.Context, resourceType string) ([]*types.ClassificationModule, error) {
	tid, e := tenant(ctx)
	if e != nil {
		return nil, e
	}
	mods, e := s.repo.ListModules(ctx, tid, resourceType)
	if e != nil {
		return nil, e
	}
	for _, m := range mods {
		opts, e := s.repo.ListOptions(ctx, tid, m.ID, resourceType)
		if e != nil {
			return nil, e
		}
		// Build ID index with pointer references
		by := make(map[string]*types.ClassificationOption, len(opts))
		for _, option := range opts {
			by[option.ID] = option
		}
		// Attach children to parents using pointers
		for _, option := range opts {
			if option.ParentID != nil {
				if parent := by[*option.ParentID]; parent != nil {
					// Store as pointer to preserve nested children
					if parent.Children == nil {
						parent.Children = make([]types.ClassificationOption, 0)
					}
					parent.Children = append(parent.Children, *option)
				}
			}
		}
		// Recursively copy tree to ensure all nested children are included
		var copyTree func(*types.ClassificationOption) types.ClassificationOption
		copyTree = func(node *types.ClassificationOption) types.ClassificationOption {
			copy := *node
			if len(node.Children) > 0 {
				copy.Children = make([]types.ClassificationOption, len(node.Children))
				for i := range node.Children {
					// Recursively copy each child and its descendants
					childPtr := by[node.Children[i].ID]
					if childPtr != nil {
						copy.Children[i] = copyTree(childPtr)
					} else {
						copy.Children[i] = node.Children[i]
					}
				}
			}
			return copy
		}
		// Collect root nodes and deep copy
		m.Options = make([]types.ClassificationOption, 0)
		for _, option := range opts {
			if option.ParentID == nil {
				m.Options = append(m.Options, copyTree(option))
			}
		}
	}
	return mods, nil
}
func (s *classificationService) CreateModule(ctx context.Context, resourceType string, name string, sortOrder int) (*types.ClassificationModule, error) {
	tid, e := tenant(ctx)
	if e != nil {
		return nil, e
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("module name is required")
	}
	v := &types.ClassificationModule{ID: uuid.NewString(), TenantID: tid, ResourceType: resourceType, Name: name, SortOrder: sortOrder, Enabled: true}
	return v, s.repo.CreateModule(ctx, v)
}
func (s *classificationService) UpdateModule(ctx context.Context, resourceType string, id string, name *string, sortOrder *int, enabled *bool) (*types.ClassificationModule, error) {
	tid, e := tenant(ctx)
	if e != nil {
		return nil, e
	}
	v, e := s.repo.GetModule(ctx, tid, id, resourceType)
	if e != nil {
		return nil, e
	}
	if name != nil {
		v.Name = strings.TrimSpace(*name)
	}
	if sortOrder != nil {
		v.SortOrder = *sortOrder
	}
	if enabled != nil {
		v.Enabled = *enabled
	}
	return v, s.repo.UpdateModule(ctx, v)
}
func (s *classificationService) DeleteModule(ctx context.Context, resourceType string, id string) error {
	tid, e := tenant(ctx)
	if e != nil {
		return e
	}
	return s.repo.DeleteModule(ctx, tid, id, resourceType)
}
func (s *classificationService) CreateOption(ctx context.Context, resourceType string, moduleID string, parentID *string, name string, sortOrder int) (*types.ClassificationOption, error) {
	tid, e := tenant(ctx)
	if e != nil {
		return nil, e
	}
	if _, e = s.repo.GetModule(ctx, tid, moduleID, resourceType); e != nil {
		return nil, e
	}
	if parentID != nil {
		p, e := s.repo.GetOption(ctx, tid, *parentID)
		if e != nil || p.ModuleID != moduleID {
			return nil, errors.New("parent option is invalid")
		}
	}
	v := &types.ClassificationOption{ID: uuid.NewString(), TenantID: tid, ModuleID: moduleID, ResourceType: resourceType, ParentID: parentID, Name: strings.TrimSpace(name), SortOrder: sortOrder, Enabled: true}
	if v.Name == "" {
		return nil, errors.New("option name is required")
	}
	createErr := s.repo.CreateOption(ctx, v)
	if createErr != nil && strings.Contains(createErr.Error(), "classification_options_module_id_parent_id_name_key") {
		return nil, apperrors.NewConflictError("同级分类名称已存在")
	}
	return v, createErr
}
func (s *classificationService) UpdateOption(ctx context.Context, resourceType string, id string, name *string, sortOrder *int, enabled *bool) (*types.ClassificationOption, error) {
	tid, e := tenant(ctx)
	if e != nil {
		return nil, e
	}
	v, e := s.repo.GetOption(ctx, tid, id)
	if e != nil {
		return nil, e
	}
	// Verify resource type matches
	if v.ResourceType != resourceType {
		return nil, errors.New("resource type mismatch")
	}
	if name != nil {
		v.Name = strings.TrimSpace(*name)
	}
	if sortOrder != nil {
		v.SortOrder = *sortOrder
	}
	if enabled != nil {
		v.Enabled = *enabled
	}
	return v, s.repo.UpdateOption(ctx, v)
}
func (s *classificationService) DeleteOption(ctx context.Context, resourceType string, id string) error {
	tid, e := tenant(ctx)
	if e != nil {
		return e
	}
	// Verify resource type before deletion
	opt, e := s.repo.GetOption(ctx, tid, id)
	if e != nil {
		return e
	}
	if opt.ResourceType != resourceType {
		return errors.New("resource type mismatch")
	}
	return s.repo.DeleteOption(ctx, tid, id)
}
func (s *classificationService) ValidateSelections(ctx context.Context, resourceType string, selections map[string][]string) error {
	tid, e := tenant(ctx)
	if e != nil {
		return e
	}
	for moduleID, ids := range selections {
		if _, e = s.repo.GetModule(ctx, tid, moduleID, resourceType); e != nil {
			return e
		}
		for _, id := range ids {
			o, e := s.repo.GetOption(ctx, tid, id)
			if e != nil || o.ModuleID != moduleID || o.ResourceType != resourceType {
				return errors.New("classification option is invalid")
			}
		}
	}
	return nil
}
