# Feature Specification: Xelora Skill Usage Governance

**Feature Branch**: `001-skill-usage-governance`

**Created**: 2026-07-08

**Status**: Draft

**Input**: User description: "Establish the current Xelora skill usage governance for AI-assisted development, covering when to use superpowers, spec-kit, project-local skills, documentation synchronization, and submission rules."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start Work With The Right Skill Path (Priority: P1)

As a contributor working on Xelora secondary development, I want a single documented rule set for which skill package to use at each stage of work, so that I can start research, planning, implementation, and release tasks without guessing or relying on chat history.

**Why this priority**: If the entry workflow is unclear, every later artifact becomes inconsistent and contributors will either skip useful skills or overuse expensive ones.

**Independent Test**: A contributor who has not participated in prior discussions can read the governance spec and correctly choose the required skill path for a new coding, planning, or documentation task.

**Acceptance Scenarios**:

1. **Given** a new development task, **When** the contributor checks the governance spec, **Then** the spec identifies which skill family is mandatory, recommended, or optional for that task type.
2. **Given** a task that spans multiple stages, **When** the contributor follows the governance spec, **Then** the spec states the handoff order between discovery, specification, planning, implementation, and release work.
3. **Given** a skill is unavailable or cannot be executed, **When** the contributor consults the governance spec, **Then** the spec defines the fallback behavior and required documentation of the exception.

---

### User Story 2 - Keep Governance And Repo Docs In Sync (Priority: P2)

As a maintainer, I want governance changes to have a documented sync rule for README and secondary-development docs, so that contributors can find current conventions from the repository itself instead of depending on informal explanations.

**Why this priority**: Governance that is not discoverable from the repo quickly becomes stale and stops being enforceable across machines and contributors.

**Independent Test**: After a governance update is added, a reviewer can verify from repository files alone where the current rules live and whether top-level indexes point to them.

**Acceptance Scenarios**:

1. **Given** a new or changed governance document, **When** it is submitted, **Then** the repository index documentation is updated to expose that document from a stable entry point.
2. **Given** a contributor opens the repository root documentation, **When** they look for customization or development guidance, **Then** they can navigate to the current governance spec without needing external instructions.

---

### User Story 3 - Review Submissions For Governance Compliance (Priority: P3)

As a reviewer, I want submissions to carry clear governance artifacts and boundaries, so that I can confirm what skill workflow was followed and ensure governance changes are not mixed with unrelated repository noise.

**Why this priority**: Review quality drops when governance work is bundled with unrelated edits or when reviewers cannot tell whether required process artifacts exist.

**Independent Test**: A reviewer can inspect the submitted files and determine whether the work includes the expected governance artifacts, README sync, and exception notes without replaying the original conversation.

**Acceptance Scenarios**:

1. **Given** a governance-related submission, **When** a reviewer inspects the changed files, **Then** the required governance artifacts and documentation links are present.
2. **Given** the repository contains unrelated dirty changes, **When** governance work is submitted, **Then** the governance submission remains isolated from unrelated modifications.

---

### Edge Cases

- A task matches more than one skill family, such as requiring both specification governance and implementation execution.
- A named skill is missing, unavailable on the current machine, or blocked by environment limitations.
- A contributor makes a small emergency fix and needs a lighter governance path without losing traceability.
- The repository already contains unrelated dirty files when governance artifacts are being prepared for submission.
- Governance rules change after existing design documents were already linked from README.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The governance set MUST define the canonical skill workflow for the main work stages used in this repository, including discovery, specification, planning, implementation, release, and review.
- **FR-002**: The governance set MUST state when `superpowers`, `spec-kit`, and project-local skills are mandatory, recommended, optional, or not applicable.
- **FR-003**: The governance set MUST define a fallback rule for cases where a required skill cannot be executed, including how the contributor records the exception and the substitute path taken.
- **FR-004**: The governance set MUST describe the minimum required artifacts for governance-controlled work, including which outputs are expected from specification work and which outputs are expected from implementation or release work.
- **FR-005**: The governance set MUST define how documentation indexes are kept synchronized, including a rule that repository entry-point documentation references current governance and newly added design or planning documents.
- **FR-006**: The governance set MUST define submission boundaries so governance artifacts can be committed or reviewed without being mixed with unrelated dirty worktree changes.
- **FR-007**: The governance set MUST be written in a form that allows a reviewer to verify compliance from repository artifacts alone, without requiring access to prior chat conversations.
- **FR-008**: The governance set MUST identify the allowed lightweight path for urgent or low-complexity work while preserving traceability and documentation expectations.
- **FR-009**: The governance set MUST remain understandable to non-authors, including contributors joining from a different machine or client.

### Key Entities

- **Skill Family**: A governed category of capability such as `superpowers`, `spec-kit`, or project-local skills; includes scope, trigger conditions, and fallback rules.
- **Work Stage**: A distinct phase of repository work such as discovery, specification, planning, implementation, review, or release.
- **Governance Artifact**: A repository file or submission output that proves the expected process was followed, such as a spec, plan, checklist, README index update, or exception note.
- **Exception Record**: A concise explanation of why a required skill path could not be followed and what alternative path was used instead.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A contributor can identify the correct required skill path for a new repository task within 10 minutes using repository documentation alone.
- **SC-002**: 100% of governance-controlled work types in scope have a documented workflow path and expected artifact set.
- **SC-003**: A reviewer can determine from the changed files alone whether README synchronization and governance artifacts were included for a governance update.
- **SC-004**: New governance or design documents become reachable from a stable repository entry point in no more than two navigation steps.
- **SC-005**: Governance submissions can be reviewed without unrelated file noise in all cases where unrelated dirty changes already exist in the worktree.

## Assumptions

- This governance feature targets contributors performing Xelora secondary development with AI assistance, not end users configuring runtime skills inside the product UI.
- Existing customization documents and the repository README remain the primary discovery surfaces for project conventions.
- The initial governance scope covers repository workflow and submission behavior; it does not yet define the full runtime sandbox architecture for web agents.
- The project may continue to reuse mature upstream or third-party modules with minimal adaptation, so governance should bias toward orchestration and documentation rather than invasive rewrites.
