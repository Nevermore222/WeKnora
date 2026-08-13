# TOIN/FS CL Technical-Point Verification Code Generation Design

**Date:** 2026-08-13

**Status:** Approved design

**Scope:** A new Xelora agent and n8n workflow that generate one complete CL/COBOL/SF verification-code set directly from one CL technical point.

## 1. Objective

Replace the intermediate two-dimensional test-case generation step for this path. The new workflow reads CL technical-point evidence directly from `analyzes.tech_point`, samples source rows, sends one technical point to a dedicated Xelora agent, and writes one complete CL/COBOL/SF file set.

The existing technical-point table workflow and the existing `CL检证用例代码生成智能体-TOIN&FS` remain unchanged. The new path is independent so its source-driven coverage rules cannot regress the existing case-table-driven generator.

## 2. Supported Requests

The webhook accepts these modes:

1. `project_id + command_id + tech_point_no`: generate one specified technical point.
2. `project_id + command_id`: generate every not-yet-completed technical point under that command, one at a time.

`project_id` is mandatory and only these projects are accepted:

| project_id | Project | Program prefix |
|---|---|---|
| 1 | TOIN | `CO` |
| 3 | Fushan | `CF` |

Any other project ID fails before querying or allocating a program number. The `CT` prefix is prohibited throughout prompts, generated code, filenames, validation, and saved metadata.

Only `language = CL` is supported. `MOVE` and `IF` commands remain excluded, matching the current technical-point workflow.

## 3. Processing Unit

One `TECH_POINT_NO` is one atomic generation unit and produces exactly one file set:

- `{PROGRAM_BASE}C{FILE_SEQ}.CL`
- `{PROGRAM_BASE}B{FILE_SEQ}.COBOL`
- `{PROGRAM_BASE}F{FILE_SEQ}.SF`

Different structures found among that technical point's sampled sources are implemented as separate verification cases inside the same CL program. A technical point is never split across multiple generated file sets.

## 4. Reference Sampling

The workflow retains the existing maximum-five sampling rule:

1. Group source rows by technical point.
2. Sort PGM IDs deterministically.
3. Select one row from each PGM bucket before selecting a second row from a bucket.
4. Continue round-robin until five rows are selected or no rows remain.

The sampling stage does not try to exhaust every source structure. Only the selected rows are provided to the agent. If fewer than five rows exist, all available rows are sent.

Each reference includes at least:

- `source_tech_point_id` (`tech_point.id`)
- `command_id`
- `command_name`
- `language`
- `tech_point_no`
- `sub_category`
- `pgmid`
- `source_content`
- `variable_declaration`
- `command_code` when present

## 5. Structure Deduplication

The dedicated agent deduplicates only within the sampled references. Two references are duplicates when both conditions hold:

1. Their normalized `variable_declaration` values are identical.
2. Their normalized SOURCE skeletons are identical.

SOURCE skeleton comparison preserves:

- command name;
- parameter names;
- parameter order;
- value categories such as variable, numeric constant, string constant, system constant, and resource/name literal;
- continuation and nesting structure where semantically meaningful.

It ignores:

- variable identifiers such as `?AFILE2` versus `?CFILE2`;
- concrete numeric values such as `BF-32` versus `BF-51`;
- concrete values within the same value category.

If either the variable declaration or SOURCE skeleton differs, the references are separate structures. Deduplication never combines variable definitions across structures. Every retained structure carries all contributing `source_tech_point_id` and PGM IDs for traceability.

## 6. Verification-Case Coverage

For every retained structure, the agent generates:

- exactly one representative normal case; and
- two or three applicable boundary-value cases.

The agent generates only `正常系` and `境界値`. It must not generate abnormal, combination, performance, permission, missing-resource, or unrelated cases.

Boundary cases must be supported by the sampled SOURCE, its variable declaration, or `Manual_ASP`. They test relevant limits such as minimum, maximum, just-inside, or just-outside a valid range only when those values are executable and meaningful for the structure. The agent must not invent unsupported limits or error codes.

Identical structures do not produce repeated case groups. The number of generated groups is the number of unique structures after deduplication, not the number of sampled references.

## 7. Dedicated Xelora Agent

Create a new agent specifically for direct CL technical-point code generation. Do not modify or reuse the deployed instance of `CL检证用例代码生成智能体-TOIN&FS`.

The new agent inherits the authoritative source-format rules and reference templates from the existing code agent:

- complete CL/COBOL/SF output;
- CL v10 fixed declarations, error handling, return handling, and result call;
- project library and file-definition conventions;
- fixed-format COBOL columns and indentation;
- valid SF/PF format ending with `/END`;
- no physical line wider than 72 display columns;
- no placeholders, omitted sections, summaries, or explanatory prose;
- Japanese comments, messages, and literals except required identifiers and syntax.

The input contract changes from `TEST_CASES_TABLE` to a single technical-point payload containing metadata and sampled references. The agent designs cases and emits runnable source in one response.

The output contains exactly these sections in order:

1. `## TOINSRC/{PROGRAM_BASE}C{FILE_SEQ}.CL`
2. `## TOINSRC/{PROGRAM_BASE}B{FILE_SEQ}.COBOL`
3. `## TOINSRC/{PROGRAM_BASE}F{FILE_SEQ}.SF`

Each heading is followed immediately by one complete fenced source block.

## 8. CL Traceability Comments

Trace metadata is written only in the CL file. COBOL and SF remain clean reference-template implementations.

Each generated case begins with CL comments containing:

- full `TECH_POINT_NO`;
- every contributing PGM ID for that structure;
- every contributing `source_tech_point_id` for that structure;
- a deterministic structure ID such as `S01`;
- a case number;
- `TEST_TYPE=正常系` or `TEST_TYPE=境界値`.

Long values, especially the technical-point number, are continued across multiple valid CL comment lines so every physical line remains within 72 display columns. No trace value may be truncated.

## 9. Program Number Allocation

Program identifiers have the form `{PREFIX}{BASE_NO}{ROLE}{FILE_SEQ}`:

- prefix: `CO` for project 1, `CF` for project 3;
- base number: three decimal digits;
- role: `C`, `B`, or `F`;
- file sequence: two decimal digits.

Examples:

- `CO001C01`, `CO001B01`, `CO001F01`
- `CF001C01`, `CF001B01`, `CF001F01`

Allocation is persisted before agent execution. A retry reuses the same assignment. Allocation is concurrency-safe through a database uniqueness constraint and transaction. Once sequence `99` is consumed for a base number, the allocator advances the three-digit base and restarts the sequence at `01`.

The allocation is unique within a project. Exhaustion of the representable namespace fails explicitly without overwriting a prior file.

## 10. Persistence Model

Create two dedicated tables under `analyzes`:

- `analyzes.cl_tech_point_code_generations`
- `analyzes.cl_tech_point_code_generation_sources`

### Technical-point generation table

One row per `project_id + language + command_id + tech_point_no`. It stores:

- generation status (`allocated`, `generating`, `completed`, `failed`);
- command and technical-point metadata;
- allocated prefix, base number, sequence, and program identifiers;
- agent ID and generation run ID;
- CL, COBOL, and SF paths;
- normalization result;
- timestamps and last error.

A completed unique row prevents a second file set for the same technical point. An allocated or failed row enables retry with the same program identifiers.

### Source-detail table

One row per selected source reference for a generation record. It stores:

- `source_tech_point_id`;
- PGM ID;
- SOURCE and variable-declaration snapshots;
- computed structure fingerprint and structure ID;
- selection order;
- generation run ID and timestamps.

Both the technical-point completion state and its selected source rows become final only after all three files are saved and normalized successfully. The final update occurs in one database transaction. Failed runs keep the allocation and failure diagnostics but do not mark the technical point completed and do not create completed source-detail records.

## 11. Workflow Data Flow

The new n8n workflow performs these stages:

1. Parse and validate webhook parameters.
2. Resolve project prefix and reject unsupported projects.
3. Query eligible, not-completed CL technical points.
4. Apply the existing PGM-first, maximum-five reference selector.
5. Allocate or load the persistent program identifier for every technical point.
6. Process technical points through Split In Batches with `Done` on output 0 and `Loop` on output 1.
7. Build a structured Xelora request containing one technical point and its references.
8. Create a Xelora session and call the new agent.
9. Assemble SSE events and require a successful completion event.
10. Extract and validate exactly three source sections.
11. Create the technical-point output directory and save the three UTF-8 staging files.
12. Invoke the existing normalization service for Shift-JIS and CRLF conversion.
13. Verify all three expected files were normalized successfully.
14. In one transaction, mark the technical point completed and persist its selected source details and paths.
15. Continue the loop and return a summary containing successful and failed technical points and file paths.

One technical-point failure must not mark that point complete. In command-wide mode, the workflow records that failure and continues with later technical points. The final response is successful only when every selected technical point succeeds; otherwise it returns a partial-failure status with separate success and failure lists. No failed point is silently reported as successful.

## 12. File Layout

Reuse the existing container and host roots:

- container: `/home/n8n/.n8n/verification_cases_code`
- normalization host mapping: `/data/n8n/data/verification_cases_code`

Use a technical-point-specific directory below the project/ASP/CL hierarchy. Its sanitized name includes command identity, technical-point identity or stable digest, and program identifier so two technical points cannot share a destination. The response returns all exact file paths.

## 13. Validation Gates

Before any file is saved, validate:

- exactly three required headings and code blocks exist;
- filenames exactly match the allocated program identifiers;
- all three code blocks are non-empty and contain no placeholders or ellipses;
- no `CT` program identifier appears;
- the CL file contains the full technical-point number and trace metadata for every retained structure;
- every retained structure has one normal and two or three boundary cases;
- only normal and boundary test types appear;
- every physical line is at most 72 display columns;
- CL declarations and result/error handling follow the inherited template;
- every COBOL non-empty line obeys fixed-format source columns;
- SF is valid and ends with `/END`.

After normalization, require exactly the three expected paths to return with status `ok`. Any validation, save, or normalization failure prevents completion registration.

## 14. Testing and Verification

Automated tests cover:

- PGM-first maximum-five selection;
- fewer-than-five behavior;
- supported and unsupported project prefixes;
- deterministic program allocation, retry reuse, sequence rollover, and concurrency uniqueness;
- structure fingerprints that ignore identifiers/concrete values but detect parameter/order/type and variable-declaration differences;
- direct agent prompt invariants, including only normal/boundary coverage and CL-only trace metadata;
- SSE assembly and missing-complete failure;
- three-section extraction and filename validation;
- 72-display-column validation;
- COBOL fixed-column validation;
- success transaction behavior and no-completion behavior for failed save/normalization;
- n8n Split In Batches output order.

A live smoke test creates a dedicated Xelora session for a representative technical point containing duplicate and distinct source structures. It must produce a valid three-file set, deduplicate the duplicate structure, generate one normal and two or three boundary cases per retained structure, and include complete CL trace comments.

## 15. Deliverables

- database migration for allocation/generation and selected-source records;
- new Xelora agent upsert/configuration script;
- agent validation and live smoke-test scripts;
- reusable structure-normalization and reference-selection tests;
- new n8n workflow JSON using the embedded Xelora API key and dedicated agent ID;
- workflow validator covering endpoints, branch order, project prefixes, persistence gates, file names, and forbidden legacy artifacts;
- design and implementation documentation.
