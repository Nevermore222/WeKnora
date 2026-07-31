package tools

import (
	"context"
	"encoding/json"

	"github.com/Tencent/Xelora/internal/piruntime"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/utils"
)

type workspaceReadInput struct {
	Path string `json:"path" jsonschema:"Relative path inside the bound workspace"`
}

type workspaceWriteInput struct {
	Path    string `json:"path" jsonschema:"Relative path inside the bound workspace"`
	Content string `json:"content" jsonschema:"Full file content to write"`
}

type workspaceEditInput struct {
	Path    string `json:"path" jsonschema:"Relative path inside the bound workspace"`
	OldText string `json:"old_text" jsonschema:"Exact text to replace"`
	NewText string `json:"new_text" jsonschema:"Replacement text"`
}

type workspaceListInput struct {
	Path string `json:"path,omitempty" jsonschema:"Relative directory path; default is workspace root"`
}

type workspaceShellInput struct {
	Command string `json:"command" jsonschema:"Shell command to run with cwd set to the bound workspace root"`
}

// WorkspaceFSTool routes Personal cwd file/shell ops through the Pi Facade.
type WorkspaceFSTool struct {
	BaseTool
	toolName                 string
	sessionWorkspaceResolver func(sessionID string) *types.SessionWorkspaceBinding
	facade                   *piruntime.Facade
}

// NewWorkspaceReadTool reads a UTF-8 file from the bound workspace.
func NewWorkspaceReadTool() *WorkspaceFSTool {
	t := &WorkspaceFSTool{
		BaseTool: BaseTool{
			name:        ToolWorkspaceRead,
			description: "Read a UTF-8 text file from the bound local workspace (Personal).",
			schema:      utils.GenerateSchema[workspaceReadInput](),
		},
		toolName: ToolWorkspaceRead,
	}
	return t
}

func NewWorkspaceWriteTool() *WorkspaceFSTool {
	return &WorkspaceFSTool{
		BaseTool: BaseTool{
			name:        ToolWorkspaceWrite,
			description: "Write a UTF-8 text file inside the bound local workspace (Personal).",
			schema:      utils.GenerateSchema[workspaceWriteInput](),
		},
		toolName: ToolWorkspaceWrite,
	}
}

func NewWorkspaceEditTool() *WorkspaceFSTool {
	return &WorkspaceFSTool{
		BaseTool: BaseTool{
			name:        ToolWorkspaceEdit,
			description: "Replace exact text in a file inside the bound local workspace (Personal).",
			schema:      utils.GenerateSchema[workspaceEditInput](),
		},
		toolName: ToolWorkspaceEdit,
	}
}

func NewWorkspaceListTool() *WorkspaceFSTool {
	return &WorkspaceFSTool{
		BaseTool: BaseTool{
			name:        ToolWorkspaceList,
			description: "List files and directories inside the bound local workspace (Personal).",
			schema:      utils.GenerateSchema[workspaceListInput](),
		},
		toolName: ToolWorkspaceList,
	}
}

func NewWorkspaceShellTool() *WorkspaceFSTool {
	return &WorkspaceFSTool{
		BaseTool: BaseTool{
			name:        ToolWorkspaceShell,
			description: "Run a shell command with cwd set to the bound local workspace root (Personal).",
			schema:      utils.GenerateSchema[workspaceShellInput](),
		},
		toolName: ToolWorkspaceShell,
	}
}

func (t *WorkspaceFSTool) WithSessionWorkspaceResolver(resolver func(sessionID string) *types.SessionWorkspaceBinding) *WorkspaceFSTool {
	t.sessionWorkspaceResolver = resolver
	return t
}

func (t *WorkspaceFSTool) WithFacade(facade *piruntime.Facade) *WorkspaceFSTool {
	t.facade = facade
	return t
}

func (t *WorkspaceFSTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	facade := t.facade
	if facade == nil {
		facade = piruntime.DefaultFacade()
	}
	if facade == nil {
		return &types.ToolResult{Success: false, Error: piruntime.ErrRuntimeUnavailable + ": sidecar not started"}, nil
	}
	var sessionID string
	if meta, ok := ToolExecFromContext(ctx); ok {
		sessionID = meta.SessionID
	}
	var binding *types.SessionWorkspaceBinding
	if t.sessionWorkspaceResolver != nil && sessionID != "" {
		binding = t.sessionWorkspaceResolver(sessionID)
	}
	return facade.Execute(ctx, piruntime.ExecuteInput{
		SessionID: sessionID,
		ToolName:  t.toolName,
		Args:      args,
		Binding:   binding,
	})
}
