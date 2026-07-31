package workspace

import "sync/atomic"

// personalLocalFoldersEnabled is set true for Personal desktop builds so
// session binding accepts absolute local folder roots.
var personalLocalFoldersEnabled atomic.Bool

// SetPersonalLocalFoldersEnabled toggles Personal local-folder workspace mode.
func SetPersonalLocalFoldersEnabled(v bool) {
	personalLocalFoldersEnabled.Store(v)
}

// PersonalLocalFoldersEnabled reports whether absolute local folder binding is active.
func PersonalLocalFoldersEnabled() bool {
	return personalLocalFoldersEnabled.Load()
}
