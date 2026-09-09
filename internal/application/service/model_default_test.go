package service

import (
	"context"
	"testing"

	"github.com/Tencent/Xelora/internal/application/repository"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func modelDefaultTestContext(tenantID uint64, systemAdmin bool) context.Context {
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, tenantID)
	return context.WithValue(ctx, types.SystemAdminContextKey, systemAdmin)
}

func TestSetDefaultModel_ReplacesOnlySameTypeDefault(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.Model{}))

	models := []*types.Model{
		{ID: "chat-old", TenantID: 1, Name: "chat-old", Type: types.ModelTypeKnowledgeQA, Source: types.ModelSourceRemote, Status: types.ModelStatusActive, IsDefault: true},
		{ID: "chat-new", TenantID: 1, Name: "chat-new", Type: types.ModelTypeKnowledgeQA, Source: types.ModelSourceRemote, Status: types.ModelStatusActive},
		{ID: "chat-other-tenant", TenantID: 2, Name: "chat-other", Type: types.ModelTypeKnowledgeQA, Source: types.ModelSourceRemote, Status: types.ModelStatusActive, IsDefault: true},
		{ID: "embed-default", TenantID: 1, Name: "embed", Type: types.ModelTypeEmbedding, Source: types.ModelSourceRemote, Status: types.ModelStatusActive, IsDefault: true},
	}
	for _, model := range models {
		require.NoError(t, db.Create(model).Error)
	}

	svc := &modelService{repo: repository.NewModelRepository(db)}
	got, err := svc.SetDefaultModel(modelDefaultTestContext(1, true), "chat-new")
	require.NoError(t, err)
	assert.True(t, got.IsDefault)

	var reloaded []types.Model
	require.NoError(t, db.Order("id").Find(&reloaded).Error)
	defaults := make(map[string]bool, len(reloaded))
	for _, model := range reloaded {
		defaults[model.ID] = model.IsDefault
	}
	assert.False(t, defaults["chat-old"])
	assert.True(t, defaults["chat-new"])
	assert.False(t, defaults["chat-other-tenant"])
	assert.True(t, defaults["embed-default"])

	visible, err := svc.ListModels(modelDefaultTestContext(2, false))
	require.NoError(t, err)
	var sharedDefault *types.Model
	for _, model := range visible {
		if model.ID == "chat-new" {
			sharedDefault = model
			break
		}
	}
	require.NotNil(t, sharedDefault)
	assert.True(t, sharedDefault.IsDefault)
	assert.True(t, sharedDefault.IsBuiltin, "foreign global default must be presented as read-only")
}

func TestSetDefaultModel_RejectsForeignAndInactiveModels(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.Model{}))
	require.NoError(t, db.Create(&types.Model{
		ID: "builtin-chat", TenantID: 10000, Name: "builtin", Type: types.ModelTypeKnowledgeQA,
		Source: types.ModelSourceRemote, Status: types.ModelStatusActive, IsBuiltin: true,
	}).Error)
	require.NoError(t, db.Create(&types.Model{
		ID: "downloading-chat", TenantID: 1, Name: "downloading", Type: types.ModelTypeKnowledgeQA,
		Source: types.ModelSourceLocal, Status: types.ModelStatusDownloading,
	}).Error)

	svc := &modelService{repo: repository.NewModelRepository(db)}
	_, err = svc.SetDefaultModel(modelDefaultTestContext(1, true), "builtin-chat")
	assert.ErrorIs(t, err, ErrDefaultModelNotTenantOwned)

	_, err = svc.SetDefaultModel(modelDefaultTestContext(1, true), "downloading-chat")
	assert.ErrorIs(t, err, ErrDefaultModelInactive)

	_, err = svc.SetDefaultModel(modelDefaultTestContext(1, false), "downloading-chat")
	assert.ErrorIs(t, err, ErrDefaultModelRequiresSystemAdmin)
}
