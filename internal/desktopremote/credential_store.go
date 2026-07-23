package desktopremote

import (
	"context"
	"errors"
)

var ErrCredentialNotFound = errors.New("credential not found")

type CredentialStore interface {
	PutRefreshToken(ctx context.Context, profileID, userID, token string) error
	GetRefreshToken(ctx context.Context, profileID, userID string) (string, error)
	DeleteRefreshToken(ctx context.Context, profileID, userID string) error
}

func credentialTarget(profileID, userID string) string {
	return "Xelora Personal/enterprise/" + profileID + "/" + userID
}

func zeroBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
