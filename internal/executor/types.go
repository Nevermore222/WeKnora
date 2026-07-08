package executor

import "time"

type JobStatus string

const (
	JobStatusRunning   JobStatus = "running"
	JobStatusSucceeded JobStatus = "succeeded"
	JobStatusFailed    JobStatus = "failed"
)

type SkillJob struct {
	ID                 string    `json:"id"`
	SessionID          string    `json:"session_id,omitempty"`
	AssistantMessageID string    `json:"assistant_message_id,omitempty"`
	RequestID          string    `json:"request_id,omitempty"`
	ToolCallID         string    `json:"tool_call_id,omitempty"`
	SkillName          string    `json:"skill_name"`
	ScriptPath         string    `json:"script_path"`
	Args               []string  `json:"args,omitempty"`
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
	ID           string    `json:"id"`
	JobID        string    `json:"job_id"`
	SessionID    string    `json:"session_id,omitempty"`
	RelativePath string    `json:"relative_path"`
	AbsolutePath string    `json:"absolute_path"`
	ContentType  string    `json:"content_type,omitempty"`
	Size         int64     `json:"size"`
	ModifiedAt   time.Time `json:"modified_at"`
}

type SkillJobRequest struct {
	SessionID          string
	AssistantMessageID string
	RequestID          string
	ToolCallID         string
	SkillName          string
	ScriptPath         string
	Args               []string
	Input              string
}
