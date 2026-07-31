package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
)

// NormalizeLocalRoot returns a cleaned absolute path for a user-chosen folder.
func NormalizeLocalRoot(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", ErrInvalidName
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

// LocalWorkspaceID returns a stable id for a normalized absolute local folder.
// Format: local:<16-byte hex of sha256(lower(clean(path)))>
func LocalWorkspaceID(normalizedAbs string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(normalizedAbs))))
	return "local:" + hex.EncodeToString(sum[:16])
}

// LocalWorkspaceName returns a display name (basename) for a local folder root.
func LocalWorkspaceName(normalizedAbs string) string {
	base := filepath.Base(filepath.Clean(normalizedAbs))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return normalizedAbs
	}
	return base
}

// IsLocalWorkspaceID reports whether id uses the Personal local-folder prefix.
func IsLocalWorkspaceID(id string) bool {
	return strings.HasPrefix(strings.TrimSpace(id), "local:")
}
