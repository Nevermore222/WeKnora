package executor

import "time"

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
	ID         string          `json:"id"`
	SessionID  string          `json:"session_id,omitempty"`
	SkillName  string          `json:"skill_name"`
	RootPath   string          `json:"root_path"`
	WorkingDir string          `json:"working_dir"`
	Provider   string          `json:"provider"`
	Status     WorkspaceStatus `json:"status"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	LastUsedAt time.Time       `json:"last_used_at"`
}

type SkillJob struct {
	ID                 string    `json:"id"`
	WorkspaceID        string    `json:"workspace_id"`
	SessionID          string    `json:"session_id,omitempty"`
	AssistantMessageID string    `json:"assistant_message_id,omitempty"`
	RequestID          string    `json:"request_id,omitempty"`
	ToolCallID         string    `json:"tool_call_id,omitempty"`
	SkillName          string    `json:"skill_name"`
	ScriptPath         string    `json:"script_path"`
	Args               []string  `json:"args,omitempty"`
	Provider           string    `json:"provider"`
	WorkingDir         string    `json:"working_dir"`
	Status             JobStatus `json:"status"`
	ExitCode           int       `json:"exit_code"`
	StartedAt          time.Time `json:"started_at"`
	FinishedAt         time.Time `json:"finished_at,omitempty"`
	DurationMs         int64     `json:"duration_ms"`
	StdoutSummary      string    `json:"stdout_summary,omitempty"`
	StderrSummary      string    `json:"stderr_summary,omitempty"`
	ArtifactCount      int       `json:"artifact_count"`
	Error              string    `json:"error,omitempty"`
}

type ArtifactRecord struct {
	ID           string               `json:"id"`
	WorkspaceID  string               `json:"workspace_id"`
	JobID        string               `json:"job_id"`
	SessionID    string               `json:"session_id,omitempty"`
	Name         string               `json:"name"`
	RelativePath string               `json:"relative_path"`
	AbsolutePath string               `json:"absolute_path"`
	ContentType  string               `json:"content_type,omitempty"`
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
	Provider           string
	SkillName          string
	ScriptPath         string
	Args               []string
	Input              string
}
