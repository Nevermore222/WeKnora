package repository

import (
	"context"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"gorm.io/gorm"
)

type classificationRepository struct{ db *gorm.DB }

func NewClassificationRepository(db *gorm.DB) interfaces.ClassificationRepository {
	return &classificationRepository{db: db}
}
func (r *classificationRepository) ListModules(ctx context.Context, tenantID uint64, resourceType string) ([]*types.ClassificationModule, error) {
	var v []*types.ClassificationModule
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND resource_type = ?", tenantID, resourceType).Order("sort_order,id").Find(&v).Error
	return v, err
}
func (r *classificationRepository) GetModule(ctx context.Context, tenantID uint64, id string, resourceType string) (*types.ClassificationModule, error) {
	var v types.ClassificationModule
	err := r.db.WithContext(ctx).Where("tenant_id=? AND id=? AND resource_type=?", tenantID, id, resourceType).First(&v).Error
	return &v, err
}
func (r *classificationRepository) CreateModule(ctx context.Context, v *types.ClassificationModule) error {
	return r.db.WithContext(ctx).Create(v).Error
}
func (r *classificationRepository) UpdateModule(ctx context.Context, v *types.ClassificationModule) error {
	return r.db.WithContext(ctx).Save(v).Error
}
func (r *classificationRepository) DeleteModule(ctx context.Context, tenantID uint64, id string, resourceType string) error {
	return r.db.WithContext(ctx).Where("tenant_id=? AND id=? AND resource_type=?", tenantID, id, resourceType).Delete(&types.ClassificationModule{}).Error
}
func (r *classificationRepository) ListOptions(ctx context.Context, tenantID uint64, moduleID string, resourceType string) ([]*types.ClassificationOption, error) {
	var rows []types.ClassificationOption
	err := r.db.WithContext(ctx).Where("tenant_id=? AND module_id=? AND resource_type=?", tenantID, moduleID, resourceType).Order("sort_order,id").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	options := make([]*types.ClassificationOption, len(rows))
	for i := range rows {
		options[i] = &rows[i]
	}
	return options, nil
}
func (r *classificationRepository) GetOption(ctx context.Context, tenantID uint64, id string) (*types.ClassificationOption, error) {
	var v types.ClassificationOption
	err := r.db.WithContext(ctx).Where("tenant_id=? AND id=?", tenantID, id).First(&v).Error
	return &v, err
}
func (r *classificationRepository) CreateOption(ctx context.Context, v *types.ClassificationOption) error {
	return r.db.WithContext(ctx).Create(v).Error
}
func (r *classificationRepository) UpdateOption(ctx context.Context, v *types.ClassificationOption) error {
	return r.db.WithContext(ctx).Save(v).Error
}
func (r *classificationRepository) DeleteOption(ctx context.Context, tenantID uint64, id string) error {
	return r.db.WithContext(ctx).Where("tenant_id=? AND id=?", tenantID, id).Delete(&types.ClassificationOption{}).Error
}
