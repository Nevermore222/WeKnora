package executor

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Tencent/Xelora/internal/types"
)

type JobStatus string

const (
	JobStatusRunning   JobStatus = "running"
	JobStatusSucceeded JobStatus = "succeeded"
	JobStatusFailed    JobStatus = "failed"
)

type WorkspaceStatus string

const (
	WorkspaceStatusActive WorkspaceStatus = "active"
)

type ArtifactKind string

const (
	ArtifactKindMarkdown     ArtifactKind = "markdown"
	ArtifactKindSpreadsheet  ArtifactKind = "spreadsheet"
	ArtifactKindPDF          ArtifactKind = "pdf"
	ArtifactKindPresentation ArtifactKind = "presentation"
	ArtifactKindImage        ArtifactKind = "image"
	ArtifactKindArchive      ArtifactKind = "archive"
	ArtifactKindOther        ArtifactKind = "other"
)

type ArtifactPreviewState string

const (
	ArtifactPreviewAvailable   ArtifactPreviewState = "available"
	ArtifactPreviewPending     ArtifactPreviewState = "pending"
	ArtifactPreviewUnsupported ArtifactPreviewState = "unsupported"
)

type ArtifactChangeType string

const (
	ArtifactChangeCreated  ArtifactChangeType = "created"
	ArtifactChangeModified ArtifactChangeType = "modified"
)

type WorkspaceRecord struct {
	ID         string `json:"id"`
	SessionID  string `json:"session_id,omitempty"`
	SkillName  string `json:"skill_name"`
	RootPath   string `json:"root_path"`
	WorkingDir string `json:"working_dir"`
	// Provider records which replaceable execution backend handled the job while
	// the workspace identity itself remains Xelora-owned and provider-agnostic.
	Provider   string          `json:"provider"`
	Status     WorkspaceStatus `json:"status"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	LastUsedAt time.Time       `json:"last_used_at"`
}

type SkillJob struct {
	ID                 string   `json:"id"`
	WorkspaceID        string   `json:"workspace_id"`
	SessionID          string   `json:"session_id,omitempty"`
	AssistantMessageID string   `json:"assistant_message_id,omitempty"`
	RequestID          string   `json:"request_id,omitempty"`
	ToolCallID         string   `json:"tool_call_id,omitempty"`
	SkillName          string   `json:"skill_name"`
	ScriptPath         string   `json:"script_path"`
	Args               []string `json:"args,omitempty"`
	// Provider tracks the backend used for execution without changing the
	// product-facing job contract, status model, or artifact linkage.
	Provider      string    `json:"provider"`
	WorkingDir    string    `json:"working_dir"`
	Status        JobStatus `json:"status"`
	ExitCode      int       `json:"exit_code"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at,omitempty"`
	DurationMs    int64     `json:"duration_ms"`
	StdoutSummary string    `json:"stdout_summary,omitempty"`
	StderrSummary string    `json:"stderr_summary,omitempty"`
	ArtifactCount int       `json:"artifact_count"`
	Error         string    `json:"error,omitempty"`
}

type ArtifactRecord struct {
	ID           string `json:"id"`
	WorkspaceID  string `json:"workspace_id"`
	JobID        string `json:"job_id"`
	SessionID    string `json:"session_id,omitempty"`
	Name         string `json:"name"`
	RelativePath string `json:"relative_path"`
	AbsolutePath string `json:"absolute_path"`
	ContentType  string `json:"content_type,omitempty"`
	// Artifact metadata is Xelora-owned so outputs remain stable even when the
	// execution provider changes from OpenSandbox to another backend later.
	Kind         ArtifactKind         `json:"kind"`
	PreviewState ArtifactPreviewState `json:"preview_state"`
	ChangeType   ArtifactChangeType   `json:"change_type"`
	Size         int64                `json:"size"`
	ModifiedAt   time.Time            `json:"modified_at"`
}

type SkillJobRequest struct {
	SessionID          string
	AssistantMessageID string
	RequestID          string
	ToolCallID         string
	// Provider is a replaceable backend selector. It must never redefine
	// workspace identity, policy ownership, or artifact semantics.
	Provider   string
	SkillName  string
	ScriptPath string
	Args       []string
	Input      string
	// WorkspaceBinding carries the session's bound workspace, if any. When
	// present and valid, the gateway routes file outputs to the workspace
	// root instead of the skill-private base path.
	WorkspaceBinding *types.SessionWorkspaceBinding
}

// ConversationOutputMode distinguishes bound conversations from legacy
// unbound ones when the runtime resolves where default file outputs go.
type ConversationOutputMode string

const (
	ConversationOutputModeBound   ConversationOutputMode = "bound"
	ConversationOutputModeUnbound ConversationOutputMode = "unbound"
)

// ConversationOutputFailureCode is a machine-readable reason explaining why
// default file creation was blocked for a conversation turn.
type ConversationOutputFailureCode string

const (
	ConversationOutputFailureBindingMissing    ConversationOutputFailureCode = "binding_missing"
	ConversationOutputFailureWorkspaceRequired ConversationOutputFailureCode = "workspace_required"
	ConversationOutputFailureBindingInvalid    ConversationOutputFailureCode = "binding_invalid"
	ConversationOutputFailurePathEscape        ConversationOutputFailureCode = "path_escape"
	ConversationOutputFailureAccessDenied      ConversationOutputFailureCode = "access_denied"
)

// ConversationOutputContext is the resolved file-output contract for one
// execution turn within a conversation. It is derived from the session
// workspace binding and consumed by executor job preparation and artifact
// registration to keep default output ownership consistent across providers.
type ConversationOutputContext struct {
	SessionID        string                        `json:"session_id,omitempty"`
	WorkspaceID      string                        `json:"workspace_id,omitempty"`
	EffectiveRootDir string                        `json:"effective_root_path,omitempty"`
	Mode             ConversationOutputMode        `json:"mode"`
	WriteAllowed     bool                          `json:"write_allowed"`
	FailureCode      ConversationOutputFailureCode `json:"failure_code,omitempty"`
	FailureMessage   string                        `json:"failure_message,omitempty"`
}

// ResolveConversationOutputContext derives the output contract from a session
// workspace binding snapshot. A nil or unbound binding yields an unbound
// context; a non-bound status yields a blocked context with a failure code.
func ResolveConversationOutputContext(sessionID string, binding *types.SessionWorkspaceBinding) ConversationOutputContext {
	if binding == nil || binding.Status == types.SessionWorkspaceBindingStatusUnbound || binding.WorkspaceID == "" {
		return ConversationOutputContext{
			SessionID:    sessionID,
			Mode:         ConversationOutputModeUnbound,
			WriteAllowed: false,
		}
	}
	if binding.Status != types.SessionWorkspaceBindingStatusBound {
		return ConversationOutputContext{
			SessionID:      sessionID,
			WorkspaceID:    binding.WorkspaceID,
			Mode:           ConversationOutputModeBound,
			WriteAllowed:   false,
			FailureCode:    bindingFailureCodeFromStatus(binding.Status),
			FailureMessage: binding.ValidationMessage,
		}
	}
	return ConversationOutputContext{
		SessionID:        sessionID,
		WorkspaceID:      binding.WorkspaceID,
		EffectiveRootDir: binding.RootPath,
		Mode:             ConversationOutputModeBound,
		WriteAllowed:     true,
	}
}

func bindingFailureCodeFromStatus(status types.SessionWorkspaceBindingStatus) ConversationOutputFailureCode {
	switch status {
	case types.SessionWorkspaceBindingStatusInvalid:
		return ConversationOutputFailureBindingInvalid
	case types.SessionWorkspaceBindingStatusAccessDenied:
		return ConversationOutputFailureAccessDenied
	case types.SessionWorkspaceBindingStatusArchived:
		return ConversationOutputFailureBindingInvalid
	default:
		return ConversationOutputFailureBindingInvalid
	}
}

// IsWithinWorkspaceRoot checks whether a resolved path stays inside the
// conversation's effective workspace root, blocking path escapes before
// any file is written.
func (c ConversationOutputContext) IsWithinWorkspaceRoot(absPath string) bool {
	if c.Mode != ConversationOutputModeBound || c.EffectiveRootDir == "" {
		return false
	}
	root := filepath.Clean(c.EffectiveRootDir)
	target := filepath.Clean(absPath)
	lexicalRel, err := filepath.Rel(root, target)
	if err != nil || lexicalRel == ".." || strings.HasPrefix(lexicalRel, ".."+string(filepath.Separator)) {
		return false
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return os.IsNotExist(err)
	}
	realTarget, err := resolveBoundaryPath(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(realRoot, realTarget)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func resolveBoundaryPath(path string) (string, error) {
	current := path
	remaining := make([]string, 0)
	for {
		if _, err := os.Lstat(current); err == nil {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			for index := len(remaining) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, remaining[index])
			}
			return filepath.Clean(resolved), nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", os.ErrNotExist
		}
		remaining = append(remaining, filepath.Base(current))
		current = parent
	}
}
