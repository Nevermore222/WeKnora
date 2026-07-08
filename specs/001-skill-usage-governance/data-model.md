# Data Model: Executor Runtime

## ExecutorWorkspace

Represents the persistent working directory for a chat session.

Fields:

- `id`: Stable workspace identifier.
- `tenant_id`: Tenant that owns the workspace.
- `user_id`: User that initiated the workspace.
- `session_id`: Chat session bound to this workspace.
- `agent_id`: Optional agent associated with the workspace.
- `root_path`: Gateway-managed filesystem path or storage prefix.
- `status`: `active`, `archived`, `locked`, `deleted`.
- `quota_bytes`: Maximum allowed workspace size.
- `used_bytes`: Last measured workspace size.
- `created_at`, `updated_at`, `last_used_at`: Lifecycle timestamps.

Relationships:

- One workspace has many jobs.
- One workspace has many artifacts.
- One chat session has one default workspace.

Validation:

- `root_path` must be inside the configured workspace root.
- A session cannot bind to multiple active default workspaces.
- Deleted workspaces cannot accept new jobs.

## ExecutorJob

Represents one execution request.

Fields:

- `id`: Stable job identifier.
- `workspace_id`: Workspace where the job runs.
- `tenant_id`, `user_id`, `session_id`, `agent_id`: Execution context.
- `mode`: `session` or `one_off`.
- `provider`: `local`, `cubesandbox`, or future provider key.
- `command`: Command or tool entrypoint requested.
- `args`: Structured arguments.
- `env`: Approved environment variables.
- `working_dir`: Workspace-relative working directory.
- `status`: `queued`, `preparing`, `running`, `succeeded`, `failed`, `cancelled`, `timed_out`, `policy_blocked`.
- `exit_code`: Process exit code when available.
- `stdout_ref`, `stderr_ref`: Log storage references.
- `error_code`, `error_message`: Normalized failure fields.
- `created_at`, `started_at`, `finished_at`: Lifecycle timestamps.

State transitions:

```text
queued -> preparing -> running -> succeeded
queued -> preparing -> running -> failed
queued -> preparing -> running -> timed_out
queued -> policy_blocked
queued -> cancelled
running -> cancelled
```

Validation:

- `working_dir` must be relative and remain inside the workspace.
- `args` must use structured JSON, not an opaque shell string for tool calls.
- `provider` must be enabled for the tenant or deployment.

## ExecutorArtifact

Represents a real file produced or modified by a job.

Fields:

- `id`: Stable artifact identifier.
- `workspace_id`: Workspace containing the file.
- `job_id`: Job that produced or registered the artifact.
- `tenant_id`, `user_id`, `session_id`: Ownership context.
- `name`: Display filename.
- `relative_path`: Workspace-relative file path.
- `mime_type`: Detected or declared MIME type.
- `size_bytes`: File size.
- `checksum`: Optional integrity hash.
- `kind`: `markdown`, `spreadsheet`, `pdf`, `presentation`, `image`, `archive`, `other`.
- `preview_state`: `available`, `unsupported`, `failed`, `pending`.
- `created_at`: Creation timestamp.

Validation:

- `relative_path` must be inside the workspace.
- Download requires tenant/session authorization.
- Artifacts cannot point to provider-internal paths.

## ExecutionPolicy

Represents the medium-restricted execution profile.

Fields:

- `id`: Policy identifier.
- `name`: Human-readable policy name.
- `network_mode`: `disabled`, `allowlist`, `default_allow`.
- `allowed_hosts`: Optional network allowlist.
- `max_duration_seconds`: Job timeout.
- `max_output_bytes`: Log/output limit.
- `max_workspace_bytes`: Workspace quota.
- `allow_dependency_install`: Boolean.
- `allow_git_write`: Boolean.
- `blocked_path_patterns`: Path deny rules.
- `blocked_command_patterns`: Command deny rules.

Validation:

- Policy must block absolute host paths outside the workspace.
- Policy must define a timeout.
- Policy changes should be auditable.

## ProviderCapability

Represents what an execution provider can currently do.

Fields:

- `provider`: Provider key.
- `status`: `available`, `degraded`, `unavailable`.
- `supports_session_workspace`: Boolean.
- `supports_one_off_job`: Boolean.
- `supports_streaming_logs`: Boolean.
- `supports_file_mount`: Boolean.
- `supported_runtimes`: List such as `python`, `node`, `shell`, `git`.
- `last_checked_at`: Health timestamp.

Validation:

- Gateway must reject jobs whose requested capability is unavailable.
