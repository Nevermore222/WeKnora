package interfaces

import (
	"context"

	"github.com/Tencent/Xelora/internal/workspace"
)

type WorkspaceService interface {
	List(ctx context.Context) ([]*workspace.Entry, error)
	Create(ctx context.Context, input workspace.CreateInput) (*workspace.Entry, error)
	Resolve(ctx context.Context, id string) (*workspace.Entry, error)
}
