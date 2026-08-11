# Xelora COBOL Project-Scoped Variable Declaration Design

## Goal

Update the Xelora COBOL tech-point verification-case and code-generation flows so TOIN (`project_id=1`) and Fukuyama (`project_id=3`) are fully isolated. Each tech point keeps its existing `tech_point_no`, `sub_category`, and `source_content` inputs and gains `variable_declaration` as a per-tech-point authoritative reference.

Fukuyama generation must use the real variable names, levels, PIC clauses, usages, signs, and decimal positions supplied by `variable_declaration`. A Fukuyama run must never update or delete TOIN verification cases. TOIN is not regenerated as part of this change, but future TOIN runs use the same project-aware workflow and fall back to the existing inference logic when `variable_declaration` is empty.

## Confirmed Requirements

- `project_id` is mandatory. Missing, non-numeric, non-positive, or unknown project IDs fail before any query with destructive consequences.
- The complete existing tech-point input contract remains in place: `tech_point_no`, `sub_category`, and `source_content` are still passed to the agent.
- `variable_declaration` is appended as a key reference, not substituted for any existing field.
- A variable declaration belongs only to its source tech point. The agent must never borrow or merge declarations across tech points in the same batch.
- Both TOIN and Fukuyama prefer `variable_declaration` when it is present.
- An empty `variable_declaration` does not block generation. That one tech point falls back to the existing `sub_category + source_content` inference behavior.
- Fukuyama regeneration must not update, delete, or renumber TOIN rows for the same command.
- Existing Xelora agents must be backed up before their online configurations are changed.

## Project Isolation

### Request Validation

The verification-case workflow and the code-generation workflow require an explicit positive integer `project_id`. Each workflow validates that the project exists in `analyzes.projects` before querying technical points or verification cases.

There is no default project and no fallback to `project_id=1`. Invalid requests stop before deletion, numbering, insertion, agent calls, or file output.

### Tech-Point Selection

The verification-case workflow queries `analyzes.tech_point` with both `command_id` and `project_id`. It selects the existing contextual fields plus `variable_declaration` and orders deterministically by `sub_category`, `tech_point_no`, and `id`.

The current `DISTINCT ON (sub_category)` behavior is removed. Every project-scoped tech-point row is preserved so multiple technical points under one subcategory are not collapsed into a single representative row.

If the target project and command have no matching technical points, the workflow fails before deleting stored verification cases.

### Verification-Case Identity

The logical identity of a stored project verification case is:

```text
command_id + project_id + case_number
```

TOIN and Fukuyama may both have `TC0001` for the same command. Deletion, maximum case-number lookup, insertion, and conflict updates all use the project-scoped identity.

## Database Migration

Database isolation is implemented as an explicit, idempotent migration at `scripts/n8n/migrations/2026-08-11-project-scope-verification-case-key.sql`, not as an n8n workflow node.

The migration performs these steps:

1. Inspect the current constraints and indexes on `analyzes.cl_verification_cases_master_proj`.
2. Check for duplicate non-null rows by `command_id + project_id + case_number`.
3. Stop and report duplicate keys without deleting or merging data if duplicates exist.
4. Remove the legacy uniqueness rule on `command_id + case_number` when present.
5. Add a unique constraint or index on `command_id + project_id + case_number`.
6. Verify that equal case numbers can coexist across projects and remain unique inside one project.

Legacy rows with `project_id IS NULL` remain untouched. New project-aware workflows never create null-project rows.

## Per-Tech-Point Agent Input

Each technical point is serialized as an independent block. A block contains:

```text
【技術点NO】<tech_point_no>
【小分類】<sub_category>
【変数宣言】<variable_declaration, only when non-empty>
【SOURCE_CONTENT】<source_content>
```

Batching may place several blocks in one request, but blocks are independent evidence boundaries. `project_id`, batch index, command metadata, and existing knowledge-base selection remain request metadata and do not become shared variable declarations.

When `variable_declaration` is empty, the block omits the declaration body. The system prompt instructs the agent to use the legacy inference path only for that block.

## Verification-Case Agent Contract

The existing COBOL tech-point agent remains in use so its ID, knowledge-base binding, and workflow integration stay stable. Its prompt receives a higher-priority contract with these rules:

- Keep `tech_point_no`, `sub_category`, `source_content`, and `variable_declaration` semantically associated with one input block.
- Treat a non-empty `variable_declaration` as authoritative for variable names, level numbers, PIC clauses, USAGE, sign, scale, group structure, and subordinate fields.
- Use `source_content` as the authoritative tested statement or source fragment.
- Do not invent a replacement declaration when a real declaration is supplied.
- Do not use a declaration from another technical point, even when names or subcategories appear similar.
- Do not introduce a variable absent from the block by copying it from another block.
- When `variable_declaration` is empty, infer executable declarations from `sub_category` and `source_content` using the existing rules and mark the inferred declarations in `備考` as verification-design assumptions.
- Repeat the complete `技術点NO` and `小分類` in every output row.
- Keep the current Japanese-only, exact 10-column Markdown output contract.
- Make `パラメータ設定` contain the applicable declarations and the tested COBOL statement so downstream code generation does not need to guess field types.

Every output row must trace to exactly one input technical point.

## Verification-Case Workflow Changes

A new workflow version is generated as `Xelora版COBOL技术点检证用例工作流（基于tech_point）_1.1.0.json`; the existing `1.0.0` JSON remains unchanged as a rollback artifact.

The generated workflow:

- validates `project_id` and project existence;
- queries technical points by `command_id + project_id`;
- selects and transports `variable_declaration` with all existing fields;
- builds independent per-tech-point request blocks;
- fails before deletion when the source query returns no rows;
- deletes only `command_id + project_id` rows;
- calculates the maximum case number inside `command_id + project_id`;
- inserts and updates by `command_id + project_id + case_number`;
- preserves the existing SSE completion, empty-body handling, table parsing, Japanese output, and basic non-empty checks;
- does not add production semantic rewriting or cross-row declaration repair.

## Code-Generation Flow

The Xelora COBOL code-generation workflow is generated as `Xelora版COBOL检证用例代码生成工作流_1.1.0.json` rather than overwriting its current `1.0.0` JSON.

The updated flow:

- requires and validates `project_id`;
- queries verification cases only from `analyzes.cl_verification_cases_master_proj` for `command_id + project_id`;
- does not fall back to the unscoped master table;
- joins source technical points by `project_id + command_id + tech_point_no`;
- passes grouped per-tech-point `source_content` and `variable_declaration` context alongside the existing verification-case table;
- keeps `parameter_config` as the immediate code-generation specification;
- requires the code agent to preserve supplied variable names, hierarchy, PIC/USAGE, sign, scale, and source statement semantics;
- forbids re-inference or cross-tech-point declaration reuse when real declarations are supplied.

The existing COBOL code constraints remain unchanged: 60 display columns for `.COBOL/.CBL`, comment indicator in column 7, valid fixed-format indentation, case-driven `OCCURS`, Shift_JIS/CRLF normalization, and the existing AI batch aggregation and validation stage.

## Backup And Rollback

Before either online Xelora agent is updated, a backup step exports the exact current database record by tenant and agent ID. Backups are stored under `E:\Xelora\n8n\backups\xelora-cobol-project-scope\<YYYYMMDD-HHMMSS>\`. Each timestamped backup includes at least:

- agent ID and tenant ID;
- name and description;
- complete `config` JSON, including the system prompt;
- model, rerank, knowledge-base, tool, and runtime settings contained in the config;
- creation and update timestamps.

The backup is parsed after writing and compared with the source row. A missing agent, empty config, write failure, parse failure, or mismatch stops the update.

The current generated workflow JSON files are also copied into the same timestamped backup set before new versions are generated. Rollback consists of restoring the saved agent config and importing the saved workflow JSON. Database-constraint rollback is documented separately and is not executed automatically because new per-project rows may exist after migration.

## Error Handling

- Invalid or missing `project_id`: fail before any source query with destructive consequences.
- Unknown project: fail before technical-point or verification-case processing.
- No technical points for the requested project and command: fail before deletion.
- Missing `tech_point_no`: reject that source row rather than produce an untraceable case.
- Empty `variable_declaration`: use legacy inference for that technical point only.
- Empty or failed SSE response: retain the existing explicit successful-`complete` requirement and do not insert rows.
- Duplicate project-scoped keys found by migration: stop and report all conflicting keys.
- Agent backup failure: stop before updating the agent.
- Code-generation source join misses a technical point: fail that batch instead of borrowing another project's or another technical point's context.

## Verification

### Static Workflow Checks

Validators assert that both generated workflows:

- require `project_id`;
- filter every project-owned query by `project_id`;
- contain no unscoped `tech_point` representative query;
- transport `variable_declaration` with the original technical-point fields;
- contain no `ON CONFLICT (command_id, case_number)` clause;
- use the project-scoped conflict key;
- preserve existing SSE and output-format markers.

### Fixture Tests

Use the supplied Fukuyama rows as fixtures:

- `ACEPTV0000214` must keep `01 W-TIME.` with `ACCEPT W-TIME FROM TIME`.
- `ACEPTV0000217` must keep `03 WK-TENCD.` with `ACCEPT WK-TENCD FROM CSL`.
- `ADD-1L00000030V000003V0000030` must keep both `I` and `J` as `PIC 9(03)`.
- `ADD-1L00000030V000017V0000170` must keep both `I1` and `I2` as `PIC 9(03) PACKED-DECIMAL`.

Additional fixtures put similar variable names with different declarations in one batch and verify that declarations never cross technical-point boundaries. A missing-declaration fixture verifies the legacy inference path.

### Database Isolation Test

Seed TOIN and Fukuyama with the same command and `TC0001`. Regenerate Fukuyama and verify:

- both rows coexist;
- the TOIN row remains byte-for-byte unchanged;
- Fukuyama numbering and updates remain inside `project_id=3`;
- rerunning Fukuyama is idempotent.

### Live Agent Checks

After backups and agent updates:

- run the verification-case agent with a mixed Fukuyama batch;
- verify every output row maps to one `技術点NO` and uses only its declaration;
- run the code-generation agent on the resulting table and source context;
- verify declarations and statements remain faithful through generated COBOL;
- rerun the existing line-width, comment-column, indentation, and `OCCURS` checks.

## Rollout Order

1. Add and verify the backup utility.
2. Back up both current agents and both current generated workflow JSON files.
3. Add and dry-run the database migration preflight.
4. Apply the project-scoped uniqueness migration.
5. Update and validate the verification-case agent prompt.
6. Generate and validate the new verification-case workflow version.
7. Update and validate the code-generation agent prompt.
8. Generate and validate the new code-generation workflow version.
9. Run fixture, database-isolation, and live SSE tests.
10. Import and publish the new n8n workflows only after all checks pass.

TOIN verification cases are not regenerated during rollout.

## Non-Goals

- Do not regenerate existing TOIN verification cases.
- Do not alter the original Dify workflows or source agent YAML files.
- Do not create a second project-specific verification-case agent.
- Do not add production n8n logic that rewrites COBOL declarations or repairs agent semantics.
- Do not infer coverage from `tech_point_no` alone; generated declarations and executable behavior remain the evidence of semantic coverage.
- Do not touch unrelated Knavis, desktop, frontend, or deployment changes already present in the worktree.
