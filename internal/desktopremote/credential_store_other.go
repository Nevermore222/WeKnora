//go:build !windows

package desktopremote

import (
	"context"
	"sync"
)

type memoryCredentialStore struct {
	mu     sync.RWMutex
	tokens map[string]string
}

func NewCredentialStore() CredentialStore {
	return &memoryCredentialStore{tokens: map[string]string{}}
}

func (s *memoryCredentialStore) PutRefreshToken(ctx context.Context, profileID, userID, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[credentialTarget(profileID, userID)] = token
	return nil
}

func (s *memoryCredentialStore) GetRefreshToken(ctx context.Context, profileID, userID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	token, ok := s.tokens[credentialTarget(profileID, userID)]
	if !ok {
		return "", ErrCredentialNotFound
	}
	return token, nil
}

func (s *memoryCredentialStore) DeleteRefreshToken(ctx context.Context, profileID, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tokens[credentialTarget(profileID, userID)]; !ok {
		return ErrCredentialNotFound
	}
	delete(s.tokens, credentialTarget(profileID, userID))
	return nil
}
