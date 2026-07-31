package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tencent/Xelora/internal/event"
	"github.com/Tencent/Xelora/internal/logger"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/utils"
	"github.com/Tencent/Xelora/internal/workspace"
)

var workspaceMutateTool = BaseTool{
	name: ToolWorkspaceMutate,
	description: `Propose a destructive file change inside the bound local workspace (delete, rename, or move).

The user must confirm in the UI before the change is applied. Relative paths only.

## Ops
- delete: remove a file or directory (from)
- rename / move: relocate from → to (both relative to workspace root)

## Returns
- Applied / rejected / expired status after the user responds`,
	schema: utils.GenerateSchema[WorkspaceMutateInput](),
}

type WorkspaceMutateInput struct {
	Op   string `json:"op" jsonschema:"Operation: delete, rename, or move"`
	From string `json:"from" jsonschema:"Source path relative to workspace root"`
	To   string `json:"to,omitempty" jsonschema:"Destination path relative to workspace root (rename/move)"`
}

type WorkspaceMutateTool struct {
	BaseTool
	sessionWorkspaceResolver func(sessionID string) *types.SessionWorkspaceBinding
	store                    *workspace.MutationStore
}

func NewWorkspaceMutateTool() *WorkspaceMutateTool {
	return &WorkspaceMutateTool{
		BaseTool: workspaceMutateTool,
		store:    workspace.DefaultMutationStore(),
	}
}

func (t *WorkspaceMutateTool) WithSessionWorkspaceResolver(resolver func(sessionID string) *types.SessionWorkspaceBinding) *WorkspaceMutateTool {
	t.sessionWorkspaceResolver = resolver
	return t
}

func (t *WorkspaceMutateTool) WithStore(store *workspace.MutationStore) *WorkspaceMutateTool {
	t.store = store
	return t
}

func (t *WorkspaceMutateTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	if !workspace.PersonalLocalFoldersEnabled() {
		return &types.ToolResult{Success: false, Error: "workspace_mutate is only available on Personal"}, nil
	}
	var input WorkspaceMutateInput
	if err := json.Unmarshal(args, &input); err != nil {
		return &types.ToolResult{Success: false, Error: fmt.Sprintf("Failed to parse args: %v", err)}, nil
	}
	input.Op = strings.ToLower(strings.TrimSpace(input.Op))
	input.From = strings.TrimSpace(input.From)
	input.To = strings.TrimSpace(input.To)
	if input.From == "" || (input.Op != "delete" && input.Op != "rename" && input.Op != "move") {
		return &types.ToolResult{Success: false, Error: "op must be delete|rename|move and from is required"}, nil
	}

	meta, ok := ToolExecFromContext(ctx)
	if !ok || meta == nil || meta.SessionID == "" {
		return &types.ToolResult{Success: false, Error: "session context required for workspace_mutate"}, nil
	}
	if t.sessionWorkspaceResolver == nil {
		return &types.ToolResult{Success: false, Error: "workspace resolver not configured"}, nil
	}
	binding := t.sessionWorkspaceResolver(meta.SessionID)
	if binding == nil || binding.Status != types.SessionWorkspaceBindingStatusBound || strings.TrimSpace(binding.RootPath) == "" {
		return &types.ToolResult{Success: false, Error: "workspace_required: bind a local folder first"}, nil
	}

	proposal, err := t.store.Propose(workspace.MutationProposal{
		SessionID:   meta.SessionID,
		WorkspaceID: binding.WorkspaceID,
		RootPath:    binding.RootPath,
		Op:          input.Op,
		FromRel:     input.From,
		ToRel:       input.To,
	})
	if err != nil {
		return &types.ToolResult{Success: false, Error: err.Error()}, nil
	}

	if meta.EventBus != nil {
		meta.EventBus.Emit(ctx, event.Event{
			Type:      event.EventWorkspaceMutationRequired,
			SessionID: meta.SessionID,
			Data: event.WorkspaceMutationRequiredData{
				ProposalID:         proposal.ID,
				Token:              proposal.Token,
				SessionID:          meta.SessionID,
				WorkspaceID:        binding.WorkspaceID,
				RootPath:           binding.RootPath,
				Op:                 proposal.Op,
				From:               proposal.FromRel,
				To:                 proposal.ToRel,
				ExpiresAtUnix:      proposal.ExpiresAt.Unix(),
				ToolCallID:         meta.ToolCallID,
				AssistantMessageID: meta.AssistantMessageID,
			},
		})
	}

	waitCtx := ctx
	if meta.ApprovalCtx != nil {
		waitCtx = meta.ApprovalCtx
	}
	approved, reason, waitErr := t.store.Wait(waitCtx, proposal.ID)
	if waitErr != nil {
		logger.Warnf(ctx, "[Tool][WorkspaceMutate] wait failed: %v", waitErr)
		return &types.ToolResult{Success: false, Error: waitErr.Error()}, nil
	}
	if !approved {
		if reason == "" {
			reason = "mutation_rejected"
		}
		return &types.ToolResult{Success: false, Error: reason}, nil
	}
	return &types.ToolResult{
		Success: true,
		Output:  "mutation applied",
		Data: map[string]interface{}{
			"status": "applied",
			"op":     proposal.Op,
			"from":   proposal.FromRel,
			"to":     proposal.ToRel,
		},
	}, nil
}
