# Data Model: Session Workspace Binding

## SessionWorkspaceBinding

Represents the durable default workspace context for one conversation.

Fields:

- `workspace_id`: Stable workspace identifier selected by the user.
- `workspace_name`: Human-readable workspace name captured for display convenience.
- `root_path`: Canonical workspace root path or storage root used for boundary validation.
- `binding_status`: Current status enum: `bound`, `unbound`, `invalid`, `access_denied`, `archived`.
- `bound_at`: Timestamp when the binding was created.
- `bound_by_user_id`: User id that created or last updated the binding.
- `last_validated_at`: Most recent successful or failed validation timestamp.
- `validation_message`: Optional user-visible explanation when the binding is not currently usable.

Relationships:

- Belongs to exactly one `Session`.
- Feeds runtime resolution for `ConversationOutputContext`.

Validation:

- `workspace_id` is required when status is not `unbound`.
- `root_path` must resolve to a canonical workspace boundary root.
- `binding_status` must match the latest validation outcome returned by the workspace authority.

## ConversationOutputContext

Represents the resolved file-output contract for one execution turn within a conversation.

Fields:

- `session_id`: Owning conversation id.
- `workspace_binding`: Snapshot of the session binding used for this turn.
- `effective_root_path`: Directory root that default outputs must stay under.
- `mode`: `bound` or `unbound`.
- `write_allowed`: Boolean gate indicating whether default file creation is permitted.
- `failure_code`: Optional machine-readable reason such as `binding_missing`, `binding_invalid`, `path_escape`, `access_denied`.
- `failure_message`: Optional user-visible explanation.

Relationships:

- Derived from `SessionWorkspaceBinding`.
- Consumed by executor job preparation and artifact registration.

Validation:

- `effective_root_path` must be empty only when `mode=unbound`.
- `write_allowed` must be false for `invalid`, `archived`, or `access_denied` bindings.

## WorkspaceArtifactRecord

Represents a generated artifact that is traceable both to the conversation and to the bound workspace.

Fields:

- `artifact_id`: Stable artifact identifier.
- `session_id`: Owning conversation id.
- `workspace_id`: Bound workspace id used for the write.
- `relative_path`: Artifact path relative to the workspace root.
- `absolute_path`: Canonical resolved path used by the runtime.
- `producer_type`: Tool, skill, or runtime component that created the artifact.
- `created_at`: Artifact creation time.
- `status`: `created`, `registered`, `failed`.

Relationships:

- Produced within one `ConversationOutputContext`.
- Referenced by existing artifact/job history surfaces.

Validation:

- `absolute_path` must remain within `effective_root_path`.
- `workspace_id` is required when the artifact comes from a bound conversation default write.

## WorkspaceBindingValidationResult

Represents the outcome of checking whether a session's binding is still usable.

Fields:

- `workspace_id`: Target workspace id.
- `status`: `valid`, `missing`, `archived`, `access_denied`, `path_unavailable`.
- `resolved_root_path`: Canonical path if validation succeeds.
- `message`: Optional explanation for logs and UI.
- `checked_at`: Validation timestamp.

Relationships:

- Used to update `SessionWorkspaceBinding.binding_status`.
- Used by `ConversationOutputContext` resolution.

Validation:

- `resolved_root_path` must be present when `status=valid`.
- Failure statuses must include a non-empty `message`.
