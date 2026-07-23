package desktopremote

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProfileStore struct {
	db *gorm.DB
}

var profileCreateSequence atomic.Int64

func NewProfileStore(db *gorm.DB) *ProfileStore {
	return &ProfileStore{db: db}
}

func (s *ProfileStore) AutoMigrate() error {
	return s.db.AutoMigrate(&ServerProfile{})
}

func (s *ProfileStore) Create(ctx context.Context, profile ServerProfile) (*ServerProfile, error) {
	if err := prepareProfile(&profile); err != nil {
		return nil, err
	}
	now := time.Now().Add(time.Duration(profileCreateSequence.Add(1)) * time.Nanosecond)
	profile.ID = uuid.NewString()
	profile.CreatedAt = now
	profile.UpdatedAt = now
	if err := s.db.WithContext(ctx).Create(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *ProfileStore) List(ctx context.Context) ([]ServerProfile, error) {
	var profiles []ServerProfile
	err := s.db.WithContext(ctx).Order("created_at ASC, id ASC").Find(&profiles).Error
	return profiles, err
}

func (s *ProfileStore) Get(ctx context.Context, id string) (*ServerProfile, error) {
	var profile ServerProfile
	if err := s.db.WithContext(ctx).First(&profile, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *ProfileStore) Update(ctx context.Context, id string, replacement ServerProfile) (*ServerProfile, error) {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := prepareProfile(&replacement); err != nil {
		return nil, err
	}

	replacement.ID = existing.ID
	replacement.CreatedAt = existing.CreatedAt
	replacement.UpdatedAt = time.Now()
	if err := s.db.WithContext(ctx).Save(&replacement).Error; err != nil {
		return nil, err
	}
	return &replacement, nil
}

func (s *ProfileStore) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&ServerProfile{}, "id = ?", id).Error
}

func prepareProfile(profile *ServerProfile) error {
	profile.Name = strings.TrimSpace(profile.Name)
	if profile.Name == "" {
		return errors.New("server name is required")
	}
	normalized, err := NormalizeServerOrigin(profile.BaseURL, profile.AllowInsecureTransport)
	if err != nil {
		return err
	}
	profile.BaseURL = normalized
	return nil
}
