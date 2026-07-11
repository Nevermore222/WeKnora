# Feature Specification: Session Workspace Binding For New Chats

**Feature Branch**: `002-session-workspace-binding`

**Created**: 2026-07-10

**Status**: Implemented for local Docker validation

**Progress Note (2026-07-12)**: The durable session binding, frontend binding
display, runtime output routing, boundary checks, and Office/structured-file
capability bridge are implemented and validated in the local Docker stack.
Remaining adjacent work is tracked separately under browser automation and
runtime observability.

**Input**: User description: "Allow users to bind a workspace when starting a new chat so all artifacts and generated files for that conversation are created inside the bound workspace by default."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Bind A Workspace When Starting A New Chat (Priority: P1)

As a user starting a fresh conversation, I want to choose or bind a workspace before I begin, so that the conversation has a clear file context from the first message and I do not need to manually restate where outputs should go.

**Why this priority**: This is the entry point that makes the rest of the capability meaningful. If the workspace is not bound at chat creation time, later file generation remains ambiguous and error-prone.

**Independent Test**: A user can start a new chat, bind one workspace during setup, send a file-producing request, and verify that the conversation records that workspace as its default output location.

**Acceptance Scenarios**:

1. **Given** a user opens the new chat flow, **When** they choose a workspace and confirm chat creation, **Then** the new conversation is created with that workspace bound as its default working context.
2. **Given** a user has access to multiple workspaces, **When** they start a new chat, **Then** the workspace choice is explicit and the selected workspace is visible in the conversation context after creation.
3. **Given** a user starts a new chat without binding a workspace, **When** the system allows the chat to proceed, **Then** the conversation is clearly marked as unbound and file-producing actions do not silently assume a hidden location.

---

### User Story 2 - Keep Conversation Outputs Inside The Bound Workspace (Priority: P2)

As a user asking the web agent to create markdown files, reports, spreadsheets, or similar artifacts, I want outputs from that conversation to land in the bound workspace by default, so that generated content is immediately usable rather than trapped in an internal skill-only directory.

**Why this priority**: The user goal is not only selecting a workspace in the UI; it is ensuring the chat can produce real files in a user-visible working area.

**Independent Test**: After a workspace-bound conversation is created, a reviewer can trigger at least one file-producing workflow and confirm that the resulting artifact is registered and stored within the conversation's bound workspace rather than a hidden fallback area.

**Acceptance Scenarios**:

1. **Given** a conversation has a bound workspace, **When** the agent creates a new file without an explicit override path, **Then** the file is created inside that workspace under the conversation's default output rules.
2. **Given** a conversation has a bound workspace, **When** a skill or execution path returns one or more artifacts, **Then** the artifact records point back to the bound workspace for that conversation.
3. **Given** a conversation includes multiple file-producing turns, **When** the user reopens the conversation later, **Then** the same workspace binding still governs subsequent default outputs unless the binding is intentionally changed.

---

### User Story 3 - Protect Workspace Boundaries And Failure Handling (Priority: P3)

As a maintainer and workspace owner, I want conversation-bound workspaces to respect access permissions, availability checks, and clear fallback behavior, so that file generation does not cross workspace boundaries or fail in confusing ways.

**Why this priority**: Once file creation is directed into real user workspaces, access control and failure semantics become part of the product contract.

**Independent Test**: A reviewer can simulate missing access, deleted workspaces, or unavailable output paths and confirm that the system blocks unsafe writes while preserving a clear user-visible explanation.

**Acceptance Scenarios**:

1. **Given** a bound workspace is no longer accessible to the user, **When** they try to continue file-producing work in that conversation, **Then** the system prevents default writes into that workspace and explains that the binding must be updated.
2. **Given** a file-producing action requests a location outside the allowed workspace scope, **When** the system evaluates the request, **Then** it rejects the write rather than escaping the conversation's permitted workspace boundary.
3. **Given** a conversation is unbound or its binding becomes invalid, **When** a file-producing task is requested, **Then** the system follows a defined recovery path instead of pretending the file was created successfully.

---

### Edge Cases

- A user starts a conversation in one workspace and later loses access to that workspace before the next file-producing turn.
- A workspace is renamed, archived, or deleted after being bound to an active conversation.
- A skill produces multiple artifacts in one turn and some requested paths are valid while others would escape the bound workspace.
- A conversation is created without a workspace because the user only wants chat at first, but later asks for real file output.
- A file-producing skill still defaults to an internal skill workspace unless the conversation binding is explicitly passed through the execution path.
- The conversation is cloned, shared, or resumed on another device and must preserve the same workspace binding semantics.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST let users bind a workspace during the new chat creation flow.
- **FR-002**: The system MUST persist the selected workspace as part of the conversation's durable context so later turns can reuse it without asking again.
- **FR-003**: The system MUST clearly indicate whether a conversation is workspace-bound or unbound after creation.
- **FR-004**: The system MUST treat the bound workspace as the default destination for conversation-generated files and artifacts when no explicit safe override is provided.
- **FR-005**: The system MUST associate artifact metadata for file-producing turns with the conversation's bound workspace.
- **FR-006**: The system MUST preserve the workspace binding when the user reopens an existing conversation or continues it on another device.
- **FR-007**: The system MUST verify that the user still has access to the bound workspace before using it for new default outputs.
- **FR-008**: The system MUST prevent conversation-driven file creation from escaping the allowed boundary of the bound workspace.
- **FR-009**: The system MUST define user-visible recovery behavior for conversations whose workspace binding is missing, invalid, or no longer accessible.
- **FR-010**: The system MUST allow file-producing skills and runtime execution paths to receive the conversation's bound workspace context through one consistent product contract.
- **FR-011**: The system MUST keep file output behavior consistent across markdown, spreadsheet, report, presentation, and similar artifact-producing workflows that rely on the conversation runtime.
- **FR-012**: The system MUST preserve compatibility for conversations that were created before workspace binding exists by treating them as unbound until a binding is explicitly added.

### Key Entities

- **Conversation Workspace Binding**: The durable relationship between one conversation and one default workspace used for later file output behavior.
- **Bound Workspace**: The user-visible workspace selected for a conversation and validated for access before new outputs are created.
- **Conversation Output Rule**: The default behavior that decides where generated files and artifacts are created when a turn does not provide a safe explicit override.
- **Workspace Artifact**: A generated file or output record that is both traceable to a conversation turn and located within the bound workspace contract.
- **Binding Validation State**: The current availability and access status of a conversation's workspace binding, used to decide whether default writes are allowed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can create a new conversation with a workspace binding in under 30 seconds without reading external documentation.
- **SC-002**: In workspace-bound conversations, 100% of default file-producing turns create or register outputs against the bound workspace rather than a hidden fallback location.
- **SC-003**: A reviewer can reopen any workspace-bound conversation and determine its current workspace binding status within one screen view.
- **SC-004**: When a workspace binding becomes invalid, users receive a clear failure or rebind path before any unsafe file write occurs.
- **SC-005**: Legacy conversations created before this feature remain usable for normal chat while clearly signaling that no default workspace output contract is active.

## Assumptions

- A conversation binds to one default workspace at a time for the initial version of this capability.
- Users may still create normal conversations without a bound workspace, but those conversations should not imply that file outputs will appear in a user workspace automatically.
- Existing runtime and artifact work should be reused where possible, with this feature extending the product contract rather than replacing current execution modules.
- The initial scope focuses on conversation-level workspace binding and default output routing, not on full interactive file browsing or arbitrary multi-root workspace management.
- Access control for workspaces already exists elsewhere in the product and this feature consumes that authority instead of redefining it.
