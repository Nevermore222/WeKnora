package interfaces

import (
	"context"

	"github.com/Tencent/Xelora/internal/workspace"
)

type WorkspaceService interface {
	List(ctx context.Context) ([]*workspace.Entry, error)
	Create(ctx context.Context, input workspace.CreateInput) (*workspace.Entry, error)
	Resolve(ctx context.Context, id string) (*workspace.Entry, error)
	ListFiles(ctx context.Context, workspaceID string, relDir string) ([]*workspace.FileEntry, error)
	PreviewFile(ctx context.Context, workspaceID string, relPath string, maxBytes int64) (*workspace.FilePreview, error)
	OpenFile(ctx context.Context, workspaceID string, relPath string) (*workspace.FileOpenResult, error)
}
