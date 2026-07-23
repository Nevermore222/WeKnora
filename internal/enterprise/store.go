package enterprise

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Store handles persistence of enterprise server configurations and resource
// cache entries in the local SQLite database.
type Store struct {
	db *gorm.DB
}

// NewStore creates a Store backed by the given GORM database handle.
func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// AutoMigrate ensures the enterprise tables exist.
func (s *Store) AutoMigrate() error {
	return s.db.AutoMigrate(&ServerConfig{}, &ResourceCacheEntry{})
}

// --- Server CRUD ---

// ListServers returns all configured enterprise servers ordered by creation time.
// Tokens are decrypted for caller use.
func (s *Store) ListServers(ctx context.Context) ([]ServerConfig, error) {
	var servers []ServerConfig
	if err := s.db.WithContext(ctx).Order("created_at ASC").Find(&servers).Error; err != nil {
		return nil, err
	}
	for i := range servers {
		plain, err := DecryptToken(servers[i].APIToken)
		if err == nil {
			servers[i].APIToken = plain
		}
		decryptLinkedFields(&servers[i])
	}
	return servers, nil
}

// GetServer returns a single server by ID with its token decrypted.
func (s *Store) GetServer(ctx context.Context, id string) (*ServerConfig, error) {
	var server ServerConfig
	if err := s.db.WithContext(ctx).First(&server, "id = ?", id).Error; err != nil {
		return nil, err
	}
	plain, err := DecryptToken(server.APIToken)
	if err == nil {
		server.APIToken = plain
	}
	decryptLinkedFields(&server)
	return &server, nil
}

// decryptLinkedFields decrypts the sensitive linked-identity fields in place.
// Empty / legacy values pass through unchanged (see DecryptToken).
func decryptLinkedFields(server *ServerConfig) {
	if plain, err := DecryptToken(server.LinkedPassword); err == nil {
		server.LinkedPassword = plain
	}
	if plain, err := DecryptToken(server.ServerJWT); err == nil {
		server.ServerJWT = plain
	}
	if plain, err := DecryptToken(server.ServerRefreshToken); err == nil {
		server.ServerRefreshToken = plain
	}
}

// CreateServer persists a new enterprise server configuration.
func (s *Store) CreateServer(ctx context.Context, name, baseURL, apiToken string, autoConnect bool) (*ServerConfig, error) {
	// Encrypt the token at rest using OS-level credential protection (DPAPI on Windows).
	encryptedToken, err := EncryptToken(apiToken)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	server := &ServerConfig{
		ID:          uuid.NewString(),
		Name:        name,
		BaseURL:     baseURL,
		APIToken:    encryptedToken,
		AutoConnect: autoConnect,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.db.WithContext(ctx).Create(server).Error; err != nil {
		return nil, err
	}
	// Return the plain token to the caller for immediate use.
	server.APIToken = apiToken
	return server, nil
}

// UpdateServer updates an existing server's mutable fields.
func (s *Store) UpdateServer(ctx context.Context, id string, name, baseURL, apiToken *string, autoConnect *bool) (*ServerConfig, error) {
	server, err := s.GetServer(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		server.Name = *name
	}
	if baseURL != nil {
		server.BaseURL = *baseURL
	}
	if apiToken != nil {
		encrypted, err := EncryptToken(*apiToken)
		if err != nil {
			return nil, err
		}
		server.APIToken = encrypted
	}
	if autoConnect != nil {
		server.AutoConnect = *autoConnect
	}
	server.UpdatedAt = time.Now()
	if err := s.db.WithContext(ctx).Save(server).Error; err != nil {
		return nil, err
	}
	return server, nil
}

// DeleteServer removes a server configuration and its cached resources.
func (s *Store) DeleteServer(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("server_id = ?", id).Delete(&ResourceCacheEntry{}).Error; err != nil {
			return err
		}
		return tx.Delete(&ServerConfig{}, "id = ?", id).Error
	})
}

// TouchServer updates the last_seen_at timestamp after a successful health check.
func (s *Store) TouchServer(ctx context.Context, id string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&ServerConfig{}).Where("id = ?", id).Update("last_seen_at", &now).Error
}

// SaveLinkedIdentity persists the auto-provisioned server identity for a server.
// The password, JWT and refresh token are encrypted at rest (DPAPI on Windows).
func (s *Store) SaveLinkedIdentity(ctx context.Context, serverID, userID, email, tenantID, password, jwt, refreshToken string) error {
	encPassword, err := EncryptToken(password)
	if err != nil {
		return err
	}
	encJWT, err := EncryptToken(jwt)
	if err != nil {
		return err
	}
	encRefresh, err := EncryptToken(refreshToken)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&ServerConfig{}).Where("id = ?", serverID).Updates(map[string]interface{}{
		"linked_user_id":       userID,
		"linked_email":         email,
		"linked_tenant_id":     tenantID,
		"linked_password":      encPassword,
		"server_jwt":           encJWT,
		"server_refresh_token": encRefresh,
		"updated_at":           time.Now(),
	}).Error
}

// ClearLinkedIdentity blanks all linked-identity fields for a server (unlink).
func (s *Store) ClearLinkedIdentity(ctx context.Context, serverID string) error {
	return s.db.WithContext(ctx).Model(&ServerConfig{}).Where("id = ?", serverID).Updates(map[string]interface{}{
		"linked_user_id":       "",
		"linked_email":         "",
		"linked_tenant_id":     "",
		"linked_password":      "",
		"server_jwt":           "",
		"server_refresh_token": "",
		"updated_at":           time.Now(),
	}).Error
}

// --- Resource Cache ---

// CacheResources replaces all cached entries for a server with fresh data.
func (s *Store) CacheResources(ctx context.Context, serverID string, entries []ResourceCacheEntry) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("server_id = ?", serverID).Delete(&ResourceCacheEntry{}).Error; err != nil {
			return err
		}
		if len(entries) == 0 {
			return nil
		}
		return tx.Create(&entries).Error
	})
}

// GetCachedResources returns all cached entries for a server, optionally filtered by type.
func (s *Store) GetCachedResources(ctx context.Context, serverID, resourceType string) ([]ResourceCacheEntry, error) {
	var entries []ResourceCacheEntry
	q := s.db.WithContext(ctx).Where("server_id = ?", serverID)
	if resourceType != "" {
		q = q.Where("resource_type = ?", resourceType)
	}
	err := q.Order("cached_at DESC").Find(&entries).Error
	return entries, err
}
