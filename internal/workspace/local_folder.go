package workspace

import (
	"fmt"
	"os"
)

// LocalFolderEntry is a Personal local-folder workspace root.
type LocalFolderEntry struct {
	ID       string
	Name     string
	RootPath string
	Status   string
}

// BindLocalFolder validates and describes an absolute local directory for Personal binding.
// On missing/permission errors it still returns a partial entry (with ID/Name/RootPath)
// alongside the error so callers can persist invalid bindings.
func BindLocalFolder(rootPath string) (*LocalFolderEntry, error) {
	norm, err := NormalizeLocalRoot(rootPath)
	if err != nil {
		return nil, err
	}
	ent := &LocalFolderEntry{
		ID:       LocalWorkspaceID(norm),
		Name:     LocalWorkspaceName(norm),
		RootPath: norm,
	}
	info, err := os.Stat(norm)
	if err != nil {
		if os.IsNotExist(err) {
			ent.Status = StatusMissing
			return ent, ErrNotFound
		}
		if os.IsPermission(err) {
			ent.Status = StatusAccessDenied
			return ent, ErrAccessDenied
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: not a directory", ErrInvalidName)
	}
	ent.Status = StatusAvailable
	return ent, nil
}
