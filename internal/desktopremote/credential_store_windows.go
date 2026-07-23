//go:build windows

package desktopremote

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2
)

var (
	advapi32       = windows.NewLazySystemDLL("advapi32.dll")
	procCredWrite  = advapi32.NewProc("CredWriteW")
	procCredRead   = advapi32.NewProc("CredReadW")
	procCredDelete = advapi32.NewProc("CredDeleteW")
	procCredFree   = advapi32.NewProc("CredFree")
)

type windowsCredential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        windows.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

type windowsCredentialStore struct{}

func NewCredentialStore() CredentialStore {
	return &windowsCredentialStore{}
}

func (s *windowsCredentialStore) PutRefreshToken(ctx context.Context, profileID, userID, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if token == "" {
		return errors.New("refresh token is required")
	}

	targetName, err := windows.UTF16PtrFromString(credentialTarget(profileID, userID))
	if err != nil {
		return err
	}
	username, err := windows.UTF16PtrFromString(userID)
	if err != nil {
		return err
	}

	blob := []byte(token)
	defer zeroBytes(blob)

	credential := windowsCredential{
		Type:               credTypeGeneric,
		TargetName:         targetName,
		CredentialBlobSize: uint32(len(blob)),
		CredentialBlob:     &blob[0],
		Persist:            credPersistLocalMachine,
		UserName:           username,
	}

	ret, _, callErr := procCredWrite.Call(uintptr(unsafe.Pointer(&credential)), 0)
	if ret == 0 {
		return fmt.Errorf("write credential: %w", callErr)
	}
	return nil
}

func (s *windowsCredentialStore) GetRefreshToken(ctx context.Context, profileID, userID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	targetName, err := windows.UTF16PtrFromString(credentialTarget(profileID, userID))
	if err != nil {
		return "", err
	}

	var credential *windowsCredential
	ret, _, callErr := procCredRead.Call(
		uintptr(unsafe.Pointer(targetName)),
		uintptr(credTypeGeneric),
		0,
		uintptr(unsafe.Pointer(&credential)),
	)
	if ret == 0 {
		if isCredentialNotFound(callErr) {
			return "", ErrCredentialNotFound
		}
		return "", fmt.Errorf("read credential: %w", callErr)
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(credential)))

	if credential.CredentialBlobSize == 0 {
		return "", nil
	}
	if credential.CredentialBlob == nil {
		return "", errors.New("credential blob is missing")
	}

	tokenBytes := make([]byte, int(credential.CredentialBlobSize))
	copy(tokenBytes, unsafe.Slice(credential.CredentialBlob, int(credential.CredentialBlobSize)))
	defer zeroBytes(tokenBytes)

	return string(tokenBytes), nil
}

func (s *windowsCredentialStore) DeleteRefreshToken(ctx context.Context, profileID, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	targetName, err := windows.UTF16PtrFromString(credentialTarget(profileID, userID))
	if err != nil {
		return err
	}

	ret, _, callErr := procCredDelete.Call(
		uintptr(unsafe.Pointer(targetName)),
		uintptr(credTypeGeneric),
		0,
	)
	if ret == 0 {
		if isCredentialNotFound(callErr) {
			return ErrCredentialNotFound
		}
		return fmt.Errorf("delete credential: %w", callErr)
	}
	return nil
}

func isCredentialNotFound(err error) bool {
	var errno syscall.Errno
	return errors.As(err, &errno) && errno == syscall.Errno(windows.ERROR_NOT_FOUND)
}
