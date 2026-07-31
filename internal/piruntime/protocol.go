package piruntime

import "encoding/json"

// ProtocolVersion is the Facade ↔ sidecar NDJSON contract version.
const ProtocolVersion = 1

// Stable error codes returned to the agent tool layer.
const (
	ErrRuntimeUnavailable = "runtime_unavailable"
	ErrPathOutOfWorkspace = "path_out_of_workspace"
	ErrProtocolMismatch   = "protocol_mismatch"
	ErrTimeout            = "timeout"
	ErrWorkspaceRequired  = "workspace_required"
	ErrBindingInvalid     = "binding_invalid"
	ErrMutationRejected   = "mutation_rejected"
	ErrUnknownTool        = "unknown_tool"
)

// Request is one NDJSON line from Go Facade to the Pi sidecar.
type Request struct {
	ProtocolVersion int             `json:"protocol_version"`
	CorrelationID   string          `json:"correlation_id"`
	SessionID       string          `json:"session_id"`
	ToolName        string          `json:"tool_name"`
	Args            json.RawMessage `json:"args"`
	Cwd             string          `json:"cwd"`
	PermissionMode  string          `json:"permission_mode,omitempty"`
	TimeoutMs       int             `json:"timeout_ms"`
}

// Response is one NDJSON line from the sidecar back to Go.
type Response struct {
	OK           bool           `json:"ok"`
	ErrorCode    string         `json:"error_code,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
	Content      string         `json:"content,omitempty"`
	Artifacts    []string       `json:"artifacts,omitempty"`
	DurationMs   int64          `json:"duration_ms,omitempty"`
	Data         map[string]any `json:"data,omitempty"`
}

// HandshakeIn is sent by the client after process start.
type HandshakeIn struct {
	Type            string `json:"type"`
	ProtocolVersion int    `json:"protocol_version"`
}

// HandshakeOut is returned by a ready sidecar.
type HandshakeOut struct {
	Type            string `json:"type"`
	ProtocolVersion int    `json:"protocol_version"`
	OK              bool   `json:"ok"`
	ErrorCode       string `json:"error_code,omitempty"`
}
