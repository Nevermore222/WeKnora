package workspace

import "sync"

// localFolderIndex maps local: workspace ids to absolute roots remembered
// after a successful Personal bind so file APIs can Resolve by id.
var localFolderIndex sync.Map

// RememberLocalFolder stores id → root for Personal local folder workspaces.
func RememberLocalFolder(id, root string) {
	if id == "" || root == "" {
		return
	}
	localFolderIndex.Store(id, root)
}

// LookupLocalFolder returns a remembered local folder root.
func LookupLocalFolder(id string) (string, bool) {
	v, ok := localFolderIndex.Load(id)
	if !ok {
		return "", false
	}
	root, _ := v.(string)
	return root, root != ""
}
