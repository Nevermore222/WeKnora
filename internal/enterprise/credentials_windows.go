//go:build windows

package enterprise

import (
	"encoding/base64"
	"fmt"
	"syscall"
	"unsafe"
)

// Windows DPAPI (Data Protection API) bindings via crypt32.dll.
// CryptProtectData encrypts data using the current Windows user's credentials,
// so only the same user on the same machine can decrypt it. This lets us store
// enterprise API tokens in SQLite without them being readable in plain text.
var (
	crypt32                     = syscall.NewLazyDLL("crypt32.dll")
	procCryptProtectData        = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData      = crypt32.NewProc("CryptUnprotectData")
	procLocalFree               = syscall.NewLazyDLL("kernel32.dll").NewProc("LocalFree")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func newDataBlob(b []byte) dataBlob {
	if len(b) == 0 {
		return dataBlob{}
	}
	return dataBlob{
		cbData: uint32(len(b)),
		pbData: &b[0],
	}
}

func (blob *dataBlob) toBytes() []byte {
	if blob.cbData == 0 || blob.pbData == nil {
		return nil
	}
	return unsafe.Slice(blob.pbData, int(blob.cbData))
}

func localFree(p *byte) {
	if p != nil {
		_, _, _ = procLocalFree.Call(uintptr(unsafe.Pointer(p)))
	}
}

// EncryptToken encrypts a plain-text API token using Windows DPAPI.
// The result is base64-encoded for safe storage in SQLite.
func EncryptToken(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	inBlob := newDataBlob([]byte(plain))
	var outBlob dataBlob

	ret, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0, // no description
		0, // no optional entropy
		0,
		0, // no prompt structure
		0, // default flags (user scope)
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if ret == 0 {
		return "", fmt.Errorf("dpapi encrypt failed: %w", err)
	}
	defer localFree(outBlob.pbData)

	return "dpapi:" + base64.StdEncoding.EncodeToString(outBlob.toBytes()), nil
}

// DecryptToken decrypts a DPAPI-encrypted token back to plain text.
// Tokens without the "dpapi:" prefix are returned as-is (legacy/plain values).
func DecryptToken(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	if len(stored) < 6 || stored[:6] != "dpapi:" {
		// Legacy or plain token — return unchanged.
		return stored, nil
	}
	raw, err := base64.StdEncoding.DecodeString(stored[6:])
	if err != nil {
		return "", fmt.Errorf("dpapi decode failed: %w", err)
	}

	inBlob := newDataBlob(raw)
	var outBlob dataBlob

	ret, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if ret == 0 {
		return "", fmt.Errorf("dpapi decrypt failed: %w", err)
	}
	defer localFree(outBlob.pbData)

	return string(outBlob.toBytes()), nil
}
