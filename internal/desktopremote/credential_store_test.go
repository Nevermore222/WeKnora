package desktopremote

import (
	"context"
	"errors"
	"testing"
)

type fakeCredentialStore struct {
	values map[string]string
}

func newFakeCredentialStore() *fakeCredentialStore {
	return &fakeCredentialStore{values: map[string]string{}}
}

func (s *fakeCredentialStore) PutRefreshToken(_ context.Context, profileID, userID, token string) error {
	s.values[credentialTarget(profileID, userID)] = token
	return nil
}

func (s *fakeCredentialStore) GetRefreshToken(_ context.Context, profileID, userID string) (string, error) {
	token, ok := s.values[credentialTarget(profileID, userID)]
	if !ok {
		return "", ErrCredentialNotFound
	}
	return token, nil
}

func (s *fakeCredentialStore) DeleteRefreshToken(_ context.Context, profileID, userID string) error {
	delete(s.values, credentialTarget(profileID, userID))
	return nil
}

func TestCredentialTargetIsStableAndScoped(t *testing.T) {
	a := credentialTarget("server-a", "user-1")
	b := credentialTarget("server-b", "user-1")
	if a == b || a != "Xelora Personal/enterprise/server-a/user-1" {
		t.Fatalf("unexpected targets %q %q", a, b)
	}
}

func TestCredentialStoreContractScopesRefreshTokens(t *testing.T) {
	var store CredentialStore = newFakeCredentialStore()
	ctx := context.Background()

	if err := store.PutRefreshToken(ctx, "server-a", "user-1", "refresh-a"); err != nil {
		t.Fatal(err)
	}
	if err := store.PutRefreshToken(ctx, "server-b", "user-1", "refresh-b"); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetRefreshToken(ctx, "server-a", "user-1")
	if err != nil || got != "refresh-a" {
		t.Fatalf("server-a token = %q, %v", got, err)
	}
	got, err = store.GetRefreshToken(ctx, "server-b", "user-1")
	if err != nil || got != "refresh-b" {
		t.Fatalf("server-b token = %q, %v", got, err)
	}

	if err := store.DeleteRefreshToken(ctx, "server-a", "user-1"); err != nil {
		t.Fatal(err)
	}
	_, err = store.GetRefreshToken(ctx, "server-a", "user-1")
	if !errors.Is(err, ErrCredentialNotFound) {
		t.Fatalf("missing credential error = %v", err)
	}
}
