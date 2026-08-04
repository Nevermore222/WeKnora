# Xelora Leadership Excel Report Design

**Date:** 2026-08-04

**Status:** Approved for implementation planning

## Objective

Create a detailed, image-led Excel report for leadership review of the current
Xelora project. The report must explain the product position, architecture,
technical strengths, representative use cases, observed validation evidence,
and current development status in a form that can be scanned quickly and then
used for technical follow-up.

The report is based on the current working tree. It must distinguish verified
capabilities, active work, and future plans. Uncommitted code or documentation
changes may be included as current-workspace evidence only when clearly labeled;
they must not be presented as a released or generally available feature.

## Audience and Story

Primary audience: project leaders and technical decision makers.

Narrative:

```text
fragmented documents
  -> document understanding and ingestion
  -> RAG retrieval and ReAct Agent reasoning
  -> Wiki and graph knowledge assets
  -> traceable, workspace-scoped, reusable outputs
```

The first sheet answers "what is Xelora and why does it matter?". The middle
sheets explain how it works and where it creates value. The final sheets provide
evidence, current status, and source boundaries.

## Deliverable

Primary artifact:

```text
Xelora-项目架构与应用价值汇报.xlsx
```

The workbook will be created outside the Git repository's source tree unless a
final delivery location is explicitly requested. The source repository and
existing dirty worktree remain untouched except for the approved design and
implementation artifacts.

## Workbook Structure

### 01_领导摘要

- Project positioning and one-sentence value proposition.
- Three capability pillars: RAG, ReAct Agent, and Wiki.
- Four evidence cards: modular replaceability, enterprise governance,
  workspace/artifact execution, and observable operations.
- One representative product image and a compact current-architecture image.

### 02_业务问题与价值闭环

- Problem-to-value storyline: fragmented information, slow retrieval,
  knowledge that is not reusable, and opaque agent execution.
- Four-step value chain: understand, retrieve, reason, and accumulate.
- Short leadership language first; technical terms are placed in supporting
  notes rather than leading paragraphs.

### 03_总体架构

- Use the existing current architecture image as the primary visual:
  `docs/xelora-current-architecture.png`.
- Explain the layers: access and product surfaces, runtime context, application
  core, execution/artifact chain, data/model infrastructure, and governance.
- Explicitly separate stable Xelora-owned contracts from replaceable providers.
- Describe the Personal/Pi workspace direction as an active design/workstream,
  not as a completed enterprise runtime feature.

### 04_核心能力与产品截图

Use existing repository screenshots with captions and source paths:

- `docs/images/qa.png`: knowledge-base question answering.
- `docs/images/agent-qa.png`: multi-step Agent reasoning and tool calls.
- `docs/images/wiki-browser.png`: generated Wiki navigation.
- `docs/images/wiki-graph.png`: graph-based knowledge exploration.
- `docs/images/langfuse.png`: agent trace and tool-level observability.
- `docs/images/settings.png`: model, parser, storage, and platform settings.

Each image receives a short caption, the capability it demonstrates, and a
note when it is a README/demo screenshot rather than a current benchmark.

### 05_亮点与技术优势

Cover only evidence-backed strengths:

- Multi-source and multi-format ingestion.
- BM25, dense, graph, rerank, and pgvector HNSW retrieval options.
- ReAct Agent, MCP tools, network search, and configurable prompts.
- Replaceable model, vector database, object storage, and deployment layers.
- Four-level tenant RBAC, resource ownership, organization sharing, and audit
  logging.
- AES-256-GCM credential protection, TLS/token-protected docreader traffic,
  SSRF-aware HTTP behavior, and Skill isolation.
- Docker, Kubernetes/Helm, private, and offline deployment paths.

Counts such as model-provider or backend totals must retain their README source
and must not be turned into quality scores without independent measurements.

### 06_使用场景与效果

Use a repeated table and image pattern:

```text
scenario -> user problem -> Xelora flow -> visible result -> evidence
```

Initial scenarios:

1. Enterprise knowledge Q&A.
2. Complex research and multi-step reasoning.
3. Automatic Wiki and knowledge graph generation.
4. Workspace-scoped Markdown, spreadsheet, report, and presentation output.
5. Private deployment with tenant and organization governance.
6. WeCom, Feishu, Slack, Telegram, Chrome, and mini-program access.

Qualitative outcomes are allowed when grounded in a screenshot, specification,
task-board entry, test record, or source file. Do not invent ROI, accuracy,
latency, adoption, or productivity percentages.

### 07_当前进展与后续规划

Three status bands:

- **已完成/已验证**: controlled Docker execution, session workspace binding,
  artifact and file capability bridge, Office/XLSX hardening, Personal
  packaging and enterprise connector work recorded on the task board.
- **进行中/当前工作区**: Personal workspace execution boundary and related
  runtime hardening; mark uncommitted changes and active design documents as
  working-tree evidence.
- **待规划**: browser automation, runtime observability/audit, client data push,
  client identity-model correction, enterprise CRUD proxy, and server-side
  resource creation, according to the current task board.

The sheet must include a small status chart derived from the task-board data,
with formulas for calculated counts and no formula errors.

### 08_证据与资料来源

- README and README_CN sections.
- `docs/customizations/TASKS.md`.
- Runtime and workspace specifications.
- Current architecture and product screenshots.
- Relevant commit hashes and working-tree status when used as evidence.
- A legend separating `已验证事实`, `当前工作区`, `设计决策`, and `路线图`.

## Visual and Layout System

- Landscape sheets with defined print areas and repeated headers.
- White canvas with deep navy structure, restrained green/purple/amber accents,
  and neutral gray evidence panels.
- Professional Chinese font with a consistent numeric font; avoid dense merged
  cells and long unbroken paragraphs.
- One dominant visual per section, with captions directly below images.
- Keep cards for repeated evidence items only; page sections remain open and
  grid-aligned rather than nested card stacks.
- Use charts for comparisons or status counts, not decorative decoration.
- Include alt/source notes in the appendix and short source labels beside
  claims when they are not self-evident.

## Evidence and Screenshot Policy

Existing README screenshots are sufficient for the first draft. After source
review, request only missing current-state images, likely:

- current workspace-binding or Personal client flow;
- a real generated artifact or report visible in the bound workspace;
- an enterprise/shared-resource view if the sharing story needs live proof.

User-provided screenshots must be free of secrets and should include the whole
relevant UI state. Screenshots supplied as evidence will be labeled with their
capture context and will not be treated as performance benchmarks by default.

## Verification and Acceptance

The report is accepted when:

1. Every major claim has a README, specification, task-board, source, test, or
   screenshot basis.
2. Completed, current-workspace, and planned items are visibly distinguished.
3. The architecture page explains ownership boundaries and does not imply that
   a provider owns Xelora product semantics.
4. Images render without clipping, distortion, overlap, or unreadable captions.
5. Sheets have stable print layouts at common desktop page widths.
6. Formula-bearing cells recalculate with zero `#REF!`, `#DIV/0!`, `#VALUE!`,
   `#N/A`, or `#NAME?` errors.
7. The final workbook can be opened without requiring the source repository to
   remain in a particular state.

## Out of Scope

- Changing product code or runtime behavior.
- Rewriting the project README in this implementation pass.
- Claiming production metrics that are not present in project evidence.
- Automatically publishing the report to external systems.
- Committing unrelated dirty-worktree files.
