# TOIN/FS CL Technical-Point Code Generation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a new Xelora agent and n8n workflow that samples one CL technical point's source evidence, generates one complete CL/COBOL/SF verification-code set, saves and normalizes the files, and records completion only after all three files succeed.

**Architecture:** Keep the existing two-dimensional-case workflow and `CL检证用例代码生成智能体-TOIN&FS` untouched. Add focused JavaScript utilities for structure fingerprints, program allocation, response extraction, and source validation; compose them into a new n8n workflow generated from a source JSON. Add two PostgreSQL tables for persistent technical-point completion, source snapshots, and retry-safe program numbers.

**Tech Stack:** Node.js ESM scripts and `node:test`, PostgreSQL migration SQL, n8n workflow JSON, Xelora `/sessions` and `/agent-chat/:session_id` APIs, existing normalization service.

---

## File Map

- Create: `scripts/n8n/migrations/2026-08-13-cl-tech-point-code-generation.sql` — allocation, completion, and selected-source tables plus indexes/constraints.
- Create: `scripts/n8n/toin-cl-tech-point-structure.mjs` — normalize SOURCE/variable declarations and compute structure fingerprints.
- Create: `scripts/n8n/toin-cl-tech-point-structure.test.mjs` — structure deduplication tests.
- Create: `scripts/n8n/toin-cl-tech-point-program-id.mjs` — project prefix and retry-safe identifier helpers.
- Create: `scripts/n8n/toin-cl-tech-point-program-id.test.mjs` — prefix, `CT` rejection, and sequence tests.
- Create: `scripts/n8n/create-xelora-toin-cl-tech-point-code-agent.mjs` — create/update the dedicated agent from the existing CL code-agent reference prompt and new technical-point contract.
- Create: `scripts/n8n/validate-xelora-toin-cl-tech-point-code-agent.mjs` — validate agent identity, prompt invariants, model limits, knowledge base, and no legacy Dify artifacts.
- Create: `scripts/n8n/smoke-test-xelora-toin-cl-tech-point-code-agent.mjs` — live representative request and three-file/traceability smoke test.
- Create: `scripts/n8n/extract-xelora-three-source-files.mjs` — pure extraction/validation of CL, COBOL, SF headings and fenced blocks.
- Create: `scripts/n8n/extract-xelora-three-source-files.test.mjs` — extraction, filename, line-width, CL trace, COBOL fixed-column, and SF `/END` tests.
- Create: `scripts/n8n/transform-toin-cl-tech-point-code-workflow.mjs` — generate the new workflow from the retained source workflow and embed the new agent credentials.
- Create: `scripts/n8n/validate-toin-cl-tech-point-code-workflow.mjs` — validate nodes, direct Xelora endpoints, project-prefix checks, batch branch order, completion gates, and forbidden legacy artifacts.
- Create: `scripts/n8n/validate-toin-cl-tech-point-code-workflow.test.mjs` — validator fixtures for branch order and persistence gates.
- Create: `n8n/检证用例代码/Xelora版TOIN_CL技术点检证用例代码生成工作流_1.0.0.json` — importable generated workflow.
- Create: `docs/superpowers/specs/2026-08-13-toin-cl-tech-point-code-generation-design.md` — approved design (already committed).
- Create: `docs/superpowers/plans/2026-08-13-toin-cl-tech-point-code-generation.md` — this implementation plan.

## Task 1: Database Migration

**Files:**
- Create: `scripts/n8n/migrations/2026-08-13-cl-tech-point-code-generation.sql`
- Test: `scripts/n8n/validate-toin-cl-tech-point-code-generation-migration.mjs`

- [ ] **Step 1: Write the failing migration validator.** Assert that the SQL contains exact tables `analyzes.cl_tech_point_code_generations` and `analyzes.cl_tech_point_code_generation_sources`, project/language/command/technical-point uniqueness, retry-preserved program IDs, source ID uniqueness, completion status, and foreign-key/index clauses.
- [ ] **Step 2: Run the validator to verify it fails.**

  Run: `node scripts/n8n/validate-toin-cl-tech-point-code-generation-migration.mjs scripts/n8n/migrations/2026-08-13-cl-tech-point-code-generation.sql`

  Expected: FAIL because the migration file does not exist.
- [ ] **Step 3: Add the migration.** Create `cl_tech_point_code_generations` with a generated UUID/run ID, `project_id`, `language`, `command_id`, `tech_point_no`, `status`, `prefix`, three-digit `base_no`, two-digit `file_seq`, program/file paths, agent ID, error, timestamps, and a unique key on `(project_id, language, command_id, tech_point_no)`. Create `cl_tech_point_code_generation_sources` with generation FK, source snapshot fields, structure fingerprint/ID, PGMID and selection order, plus unique `(generation_id, source_tech_point_id)`.
- [ ] **Step 4: Run the validator and a PostgreSQL parse check.**

  Run: `node scripts/n8n/validate-toin-cl-tech-point-code-generation-migration.mjs scripts/n8n/migrations/2026-08-13-cl-tech-point-code-generation.sql`; then pipe `BEGIN;`, the migration, and `ROLLBACK;` to `docker exec -i Xelora-postgres psql -U postgres -d Xelora -v ON_ERROR_STOP=1`.

  Expected: validator passes and PostgreSQL accepts the migration.
- [ ] **Step 5: Commit.**

  `git add -f scripts/n8n/migrations/2026-08-13-cl-tech-point-code-generation.sql scripts/n8n/validate-xelora-toin-cl-tech-point-code-generation-migration.mjs && git commit -m "feat: add CL tech-point generation records"`

## Task 2: Structure Fingerprinting and Sampling Contract

**Files:**
- Create: `scripts/n8n/toin-cl-tech-point-structure.mjs`
- Create: `scripts/n8n/toin-cl-tech-point-structure.test.mjs`
- Modify: `scripts/n8n/toin-cl-reference-selection.mjs` only if a reusable exported selector is required; do not change the existing workflow behavior.

- [ ] **Step 1: Write failing tests.** Cover: `?AFILE2` vs `?CFILE2` with identical declarations and command skeleton are equal; `BF-32` vs `BF-51` are equal; changed parameter order/type or declaration is different; source/variable normalization is deterministic; the existing PGM-first selector returns at most five rows and all rows when fewer exist.
- [ ] **Step 2: Run the tests to verify failure.**

  Run: `node --test scripts/n8n/toin-cl-tech-point-structure.test.mjs`

  Expected: FAIL with missing module/functions.
- [ ] **Step 3: Implement pure helpers.** Export `normalizeVariableDeclaration`, `normalizeSourceSkeleton`, `structureFingerprint`, `groupReferencesByStructure`, and `selectTechnicalPointReferences`. Preserve command names, parameter names/order and value categories; replace concrete variable identifiers and literal values only in the skeleton; hash the normalized declaration plus skeleton with SHA-256.
- [ ] **Step 4: Run tests and inspect exact fingerprints.**

  Expected: all tests pass; duplicate references share a fingerprint and distinct structures do not.
- [ ] **Step 5: Commit.**

  `git add scripts/n8n/toin-cl-tech-point-structure.mjs scripts/n8n/toin-cl-tech-point-structure.test.mjs && git commit -m "feat: fingerprint CL technical-point structures"`

## Task 3: Program-ID Allocation Helpers

**Files:**
- Create: `scripts/n8n/toin-cl-tech-point-program-id.mjs`
- Create: `scripts/n8n/toin-cl-tech-point-program-id.test.mjs`

- [ ] **Step 1: Write failing tests.** Assert project 1 maps to `CO`, project 3 maps to `CF`, all other IDs throw, `CT` is rejected, formatting produces `CO001C01/CO001B01/CO001F01`, sequence `99` rolls to the next base, and retry input returns the exact prior identifiers.
- [ ] **Step 2: Run tests to verify failure.**

  Run: `node --test scripts/n8n/toin-cl-tech-point-program-id.test.mjs`

  Expected: FAIL because helpers do not exist.
- [ ] **Step 3: Implement helpers.** Export `projectPrefix(projectId)`, `formatProgramFiles(prefix, baseNo, fileSeq)`, `nextProgramAllocation(baseNo, fileSeq)`, and `assertSupportedPrefix`. Keep all numeric formatting deterministic and reject overflow instead of wrapping.
- [ ] **Step 4: Run tests to verify pass.**
- [ ] **Step 5: Commit.**

  `git add scripts/n8n/toin-cl-tech-point-program-id.mjs scripts/n8n/toin-cl-tech-point-program-id.test.mjs && git commit -m "feat: add project-safe CL program identifiers"`

## Task 4: Dedicated Xelora Agent

**Files:**
- Create: `scripts/n8n/create-xelora-toin-cl-tech-point-code-agent.mjs`
- Create: `scripts/n8n/validate-xelora-toin-cl-tech-point-code-agent.mjs`
- Create: `scripts/n8n/smoke-test-xelora-toin-cl-tech-point-code-agent.mjs`

- [ ] **Step 1: Write the validator before the agent script.** Require a new agent name/ID distinct from `ea572e95-405b-43da-b8e7-70a4a22e9cce` and `d12b1c4d-686a-42da-8141-2ce5e0230d1e`; require technical-point input markers, structure-dedup rules, only `正常系`/`境界値`, CL-only trace metadata, complete three-file output, 72-column/fixed COBOL rules, `CO`/`CF` prefixes, and no Dify/environment-variable dependency.
- [ ] **Step 2: Run the validator to verify failure.**

  Expected: FAIL because the dedicated agent has not been created.
- [ ] **Step 3: Implement the upsert script.** Read the existing CL code-agent prompt/reference templates, remove the old `TEST_CASES_TABLE`-only contract, add `TECH_POINT_NO`, `SUB_CATEGORY`, `REFERENCES`, structure deduplication, normal/boundary-only coverage, and CL trace-comment rules. Persist a new agent under a new name and write its ID/KB metadata to a local output file for workflow generation. Keep the API key out of source scripts except where the user-approved workflow embedding requires it.
- [ ] **Step 4: Run the validator against the created agent.**
- [ ] **Step 5: Run the smoke test.** Send duplicate and distinct references for one technical point; require exactly three sections, no `CT`, full technical-point trace comments in CL only, one normal plus 2–3 boundaries per unique structure, and valid line/column rules.
- [ ] **Step 6: Commit scripts only.**

  `git add -f scripts/n8n/create-xelora-toin-cl-tech-point-code-agent.mjs scripts/n8n/validate-xelora-toin-cl-tech-point-code-agent.mjs scripts/n8n/smoke-test-xelora-toin-cl-tech-point-code-agent.mjs && git commit -m "feat: add dedicated CL tech-point code agent"`

## Task 5: Three-File Extraction and Validation

**Files:**
- Create: `scripts/n8n/extract-xelora-three-source-files.mjs`
- Create: `scripts/n8n/extract-xelora-three-source-files.test.mjs`

- [ ] **Step 1: Write failing tests.** Test exact CL/COBOL/SF heading extraction, fenced-block closure, filename matching, escaped pipes and multiline content staying intact, CL trace metadata completeness, only normal/boundary markers, 72 display-column limits, COBOL indicator column 7, and SF `/END`.
- [ ] **Step 2: Run tests to verify failure.**

  Run: `node --test scripts/n8n/extract-xelora-three-source-files.test.mjs`

  Expected: FAIL because extraction helpers do not exist.
- [ ] **Step 3: Implement the pure extractor/validator.** Export `extractThreeSourceFiles(markdown, expectedFiles, metadata)` and return `{cl, cobol, sf, cases, structures}` or a descriptive error. Never repair source; reject invalid output.
- [ ] **Step 4: Run tests and verify malformed output is rejected.**
- [ ] **Step 5: Commit.**

  `git add scripts/n8n/extract-xelora-three-source-files.mjs scripts/n8n/extract-xelora-three-source-files.test.mjs && git commit -m "feat: validate generated CL COBOL SF sources"`

## Task 6: Build the New n8n Workflow

**Files:**
- Create: `scripts/n8n/transform-toin-cl-tech-point-code-workflow.mjs`
- Create: `scripts/n8n/validate-toin-cl-tech-point-code-workflow.mjs`
- Create: `scripts/n8n/validate-toin-cl-tech-point-code-workflow.test.mjs`
- Create: `n8n/检证用例代码/Xelora版TOIN_CL技术点检证用例代码生成工作流_1.0.0.json`

- [ ] **Step 1: Write validator fixtures first.** Include a reversed Split In Batches fixture that fails, an unsupported project fixture that fails, a workflow without the completion transaction that fails, and a valid fixture that passes.
- [ ] **Step 2: Run validator tests to verify failure.**
- [ ] **Step 3: Implement the transformer.** Start from the existing CL code workflow only for directory, three-file save, normalization, webhook response, and Xelora session patterns. Replace its case-table query/organization with: project validation; `tech_point` query excluding MOVE/IF and completed generation rows; PGM-first five-row selector; persistent allocation query/insert; one technical point per Split In Batches item; direct technical-point request; new agent ID/API key; source extraction; three-file validation; file saves; normalization; transactional generation/source completion insert. Set output 0 to final/continuation and output 1 to the next technical-point processing node according to n8n Split In Batches semantics.
- [ ] **Step 4: Generate the workflow JSON.**

  Run: `node scripts/n8n/transform-toin-cl-tech-point-code-workflow.mjs n8n/检证用例代码/Xelora版CL检证用例代码生成工作流_TOIN&FS_1.0.0.json n8n/检证用例代码/Xelora版TOIN_CL技术点检证用例代码生成工作流_1.0.0.json`
- [ ] **Step 5: Run workflow validator tests.** Require direct `/sessions` and `/agent-chat/`, embedded approved API key, new agent ID, no Dify endpoints, no `$env`, no `CT`, project prefix guards, exactly one technical point per item, and completion only after all three save/normalize results.
- [ ] **Step 6: Commit generator, validator, and generated workflow.**

  `git add -f scripts/n8n/transform-toin-cl-tech-point-code-workflow.mjs scripts/n8n/validate-toin-cl-tech-point-code-workflow.mjs scripts/n8n/validate-toin-cl-tech-point-code-workflow.test.mjs n8n/检证用例代码/Xelora版TOIN_CL技术点检证用例代码生成工作流_1.0.0.json && git commit -m "feat: add direct CL tech-point code workflow"`

## Task 7: End-to-End Verification

**Files:**
- Verify: all files created in Tasks 1–6.
- Modify: only the specific Task 1–6 implementation or test file named by a failing verification command; do not change unrelated application files.

- [ ] **Step 1: Run all pure tests.**

  `node --test scripts/n8n/toin-cl-tech-point-structure.test.mjs scripts/n8n/toin-cl-tech-point-program-id.test.mjs scripts/n8n/extract-xelora-three-source-files.test.mjs scripts/n8n/validate-toin-cl-tech-point-code-workflow.test.mjs`

- [ ] **Step 2: Validate migration and generated workflow.**

  `node scripts/n8n/validate-xelora-toin-cl-tech-point-code-generation-migration.mjs scripts/n8n/migrations/2026-08-13-cl-tech-point-code-generation.sql`

  `node scripts/n8n/validate-toin-cl-tech-point-code-workflow.mjs n8n/检证用例代码/Xelora版TOIN_CL技术点检证用例代码生成工作流_1.0.0.json`
- [ ] **Step 3: Validate and smoke-test the dedicated Xelora agent.**

  `node scripts/n8n/validate-xelora-toin-cl-tech-point-code-agent.mjs`

  `node scripts/n8n/smoke-test-xelora-toin-cl-tech-point-code-agent.mjs`
- [ ] **Step 4: Run existing regressions.**

  `node --test scripts/n8n/toin-cl-reference-selection.test.mjs`

  `node scripts/n8n/validate-toin-cl-reference-workflow.mjs n8n/二维表/Xelora版TOIN_CL技术点检证用例二维表工作流_1.4.0.json`
- [ ] **Step 5: Run a live one-technical-point workflow smoke test.** Use `project_id=1`, one known CL `tech_point_no`, verify three files exist, names start with `CO`, all three normalize successfully, CL comments include the complete technical point/PGM/source IDs, and both generation tables contain rows only after success.
- [ ] **Step 6: Review final diff and status.** Ensure unrelated dirty-worktree files are not staged and no generated credentials are committed outside the requested workflow JSON.

## Self-Review Checklist

- Spec coverage: project 1/3 prefixes, `CT` rejection, two entry modes, one technical point per file set, maximum-five sampling, structure deduplication, normal/boundary-only cases, CL-only trace comments, retry-safe allocation, two completion tables, three-file normalization gate, and Split In Batches order are all covered above.
- Placeholder scan: no `TBD`, `TODO`, or unspecified implementation step remains.
- Type consistency: generation table/source table names, `structureFingerprint`, `formatProgramFiles`, `extractThreeSourceFiles`, workflow transformer/validator names, and test paths are consistent across tasks.
