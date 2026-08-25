package repository

import (
	"context"
	"testing"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserRepositoryCreateUserPreservesExplicitCapabilityDenies(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.User{}))

	user := &types.User{
		ID:                     "client-user-1",
		Username:               "client-user",
		Email:                  "client-user@example.com",
		PasswordHash:           "password-hash",
		IsActive:               true,
		IsClientProvisioned:    true,
		CanCreateKnowledgeBase: false,
		CanCreateAgent:         false,
		CanBindModel:           false,
	}

	repo := NewUserRepository(db)
	require.NoError(t, repo.CreateUser(context.Background(), user))

	var stored types.User
	require.NoError(t, db.First(&stored, "id = ?", user.ID).Error)
	require.True(t, stored.IsClientProvisioned)
	require.False(t, stored.CanCreateKnowledgeBase)
	require.False(t, stored.CanCreateAgent)
	require.False(t, stored.CanBindModel)
}
