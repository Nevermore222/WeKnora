//go:build !windows

package enterprise

// Non-Windows fallback: tokens are stored as-is (base64 marker prefix only).
// DPAPI is Windows-specific; on macOS/Linux a keychain integration could be
// added later. For development builds this passthrough is acceptable.

// EncryptToken returns the token unchanged on non-Windows platforms.
func EncryptToken(plain string) (string, error) {
	return plain, nil
}

// DecryptToken returns the stored token unchanged on non-Windows platforms.
func DecryptToken(stored string) (string, error) {
	if len(stored) >= 6 && stored[:6] == "dpapi:" {
		// A token encrypted on Windows cannot be decrypted here.
		return "", nil
	}
	return stored, nil
}
