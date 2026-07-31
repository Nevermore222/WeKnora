package piruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/workspace"
	"github.com/google/uuid"
)

// ExecuteInput is a Facade call from an agent tool.
type ExecuteInput struct {
	SessionID string
	ToolName  string
	Args      json.RawMessage
	Binding   *types.SessionWorkspaceBinding
	TimeoutMs int
}

// FallbackFunc runs the legacy Go executor path when Pi is not used.
type FallbackFunc func(ctx context.Context, tool string, args json.RawMessage) (*types.ToolResult, error)

// Facade routes Personal cwd tools to the Pi sidecar.
type Facade struct {
	client   Caller
	fallback FallbackFunc
}

// NewFacade constructs a Facade. client may be nil when only fallback is used.
func NewFacade(client Caller, fallback FallbackFunc) *Facade {
	return &Facade{client: client, fallback: fallback}
}

// Execute runs a workspace tool via Pi (Personal) or fallback (non-Personal).
func (f *Facade) Execute(ctx context.Context, in ExecuteInput) (*types.ToolResult, error) {
	if !workspace.PersonalLocalFoldersEnabled() {
		if f.fallback == nil {
			return &types.ToolResult{Success: false, Error: ErrRuntimeUnavailable + ": no fallback"}, nil
		}
		return f.fallback(ctx, in.ToolName, in.Args)
	}

	if in.Binding == nil || in.Binding.Status != types.SessionWorkspaceBindingStatusBound || strings.TrimSpace(in.Binding.RootPath) == "" {
		return &types.ToolResult{
			Success: false,
			Error:   ErrWorkspaceRequired + ": bind a workspace before running file tools",
		}, nil
	}
	root := strings.TrimSpace(in.Binding.RootPath)
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return &types.ToolResult{
			Success: false,
			Error:   ErrBindingInvalid + ": root_path missing",
		}, nil
	}
	if f.client == nil {
		return &types.ToolResult{Success: false, Error: ErrRuntimeUnavailable + ": sidecar not started"}, nil
	}

	timeout := in.TimeoutMs
	if timeout <= 0 {
		timeout = 60_000
	}
	callCtx := ctx
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
		defer cancel()
	}

	resp, err := f.client.Call(callCtx, Request{
		ProtocolVersion: ProtocolVersion,
		CorrelationID:   uuid.NewString(),
		SessionID:       in.SessionID,
		ToolName:        in.ToolName,
		Args:            in.Args,
		Cwd:             root,
		TimeoutMs:       timeout,
	})
	if err != nil {
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("%s: %v", ErrRuntimeUnavailable, err),
		}, nil
	}
	if !resp.OK {
		code := resp.ErrorCode
		if code == "" {
			code = ErrRuntimeUnavailable
		}
		msg := resp.ErrorMessage
		if msg == "" {
			msg = code
		}
		return &types.ToolResult{Success: false, Error: code + ": " + msg, Data: resp.Data}, nil
	}
	return &types.ToolResult{
		Success: true,
		Output:  resp.Content,
		Data:    resp.Data,
	}, nil
}
