# Xelora Excel Visual Parser Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a selectable `xlsx_visual` parser to Xelora while preserving the existing table-oriented Excel parser and selectively adapting safe parser fixes from official WeKnora `tencent-v0.8.2`.

**Architecture:** Keep the existing Python DocReader gRPC boundary and the Go ingestion pipeline as the system of record. Add worksheet classification, rendering, DrawingML extraction, deterministic manifests, and optional visual transcription in a new Python provider; return Markdown, inline image references, metadata, and optional typed source locators through the existing `Read`/`ReadStream` contracts. The ordinary `builtin` Excel path remains unchanged and is never silently rerouted.

**Tech Stack:** Python 3.10+, `openpyxl`, `pandas`, `Pillow`, `lxml`, `pytest`/`unittest`, gRPC/protobuf, Go 1.x, existing Xelora image resolver/OCR/VLM/chunking pipeline, Excel COM on Windows when available, LibreOffice headless fallback.

**Spec:** `docs/superpowers/specs/2026-10-12-xlsx-visual-parser-design.md`

## Global Constraints

- Preserve ordinary `builtin` Excel parsing for table-oriented workbooks.
- Register the explicit engine name `xlsx_visual`; do not change the default engine for `.xlsx` or `.xls`.
- Reuse Go-side image storage, OCR/VLM processing, chunking, embedding, and retrieval; the Python provider never writes to MinIO, COS, or local application storage.
- Inspect/render original workbook bytes before applying merged-cell normalization, and apply merge filling only to the table path.
- Keep `Read` and `ReadStream` backward-compatible; only append protobuf fields with new field numbers.
- Do not copy official migrations; Xelora migration numbers `000064` through `000070` are already allocated.
- Rendering uses isolated temporary directories, bounded timeouts, and process-tree cleanup; a renderer or VLM failure must produce an explicit warning, never an empty successful document.
- Every implementation task ends with focused tests and a small commit; each commit is independently revertible.

---

### Task 1: Freeze the parser backport boundary

**Files:**
- Modify: `docreader/parser/excel_parser.py`, `docreader/parser/xlsx_merge.py`, `docreader/parser/pdf_parser.py`, `docreader/parser/doc_parser.py`, `docreader/parser/docx_parser.py`, `docreader/parser/markdown_parser.py`, `docreader/parser/web_parser.py`, `docreader/parser/opendataloader_parser.py`, `docreader/utils/tempfile.py` only after reviewing the corresponding official diff.
- Test: `docreader/tests/test_excel_parser.py`, `docreader/tests/test_pdf_router.py`, `docreader/tests/test_markdown_table_util.py`.

**Interfaces:**
- Consumes: official tag `tencent-v0.8.2` at `3e8b0bfc`; current Xelora branch `feat/xlsx-visual-parser`.
- Produces: an explicit file-by-file backport list with no whole-branch merge and no migration changes.

- [ ] **Step 1: Enumerate official parser changes without applying them.**

  Run:

  ```powershell
  git diff --name-status 3e8b0bfc -- docreader/parser docreader/utils internal/infrastructure/docparser
  git diff 3e8b0bfc -- docreader/parser/excel_parser.py docreader/parser/xlsx_merge.py docreader/parser/pdf_parser.py
  ```

  Record each candidate as `adapt`, `reject`, or `defer`, with the Xelora equivalent file and an acceptance test.

- [ ] **Step 2: Keep only the approved P0 fixes.**

  The first backport batch is limited to Excel image-function filtering (`DISPIMG`/`IMAGE`), first-row header behavior, chart-safe merge preprocessing, DOCX merge/table retention, mislabeled legacy DOC detection, PDF SMask/bitmap lifetime fixes, converter process-group termination, and failed temporary-file cleanup.

- [ ] **Step 3: Verify the boundary before code changes.**

  Run `git diff --check` and `python -m pytest docreader/tests/test_excel_parser.py docreader/tests/test_pdf_router.py docreader/tests/test_markdown_table_util.py -q`.

- [ ] **Step 4: Commit the reviewable boundary.**

  ```powershell
  git add docs/superpowers/plans/2026-10-12-xlsx-visual-parser.md
  git commit -m "docs: bound official parser backports for xlsx visual work"
  ```

  Rollback point: revert this documentation-only commit if the official comparison changes; no runtime behavior is changed by this task.

### Task 2: Backport P0 parser correctness fixes

**Files:**
- Modify: `docreader/parser/excel_parser.py`
- Modify: `docreader/parser/xlsx_merge.py`
- Modify: `docreader/parser/doc_parser.py`
- Modify: `docreader/parser/docx_parser.py`
- Modify: `docreader/parser/pdf_parser.py`
- Modify: `docreader/utils/tempfile.py`
- Test: `docreader/tests/test_excel_parser.py`
- Test: `docreader/tests/test_pdf_embedded_images.py` (create from the official focused test when Xelora dependencies support it)
- Test: `docreader/tests/test_docx_merge.py` and `docreader/tests/test_docx_tables.py` (create/adapt official fixtures without changing Xelora parser contracts)
- Test: `docreader/tests/test_tempfile_context.py` (create/adapt the cleanup regression test)

**Interfaces:**
- Consumes: existing `ExcelParser.parse_into_text`, `fill_merged_cells_xlsx`, PDF/DOCX parser APIs, and Xelora test fixtures.
- Produces: behavior-compatible fixes with the same `Document` return type and no changes to Go ingestion.

- [ ] **Step 1: Add failing regression tests for each accepted P0 behavior.**

  Cover an `IMAGE`/`DISPIMG` workbook, a first-row header override, a workbook containing a chart plus merged cells, a vertical DOCX merge, mislabeled legacy DOC bytes, a PDF image with an SMask, a converter timeout, and a failed temporary-file context.

- [ ] **Step 2: Run only the new tests and confirm they fail for the expected old behavior.**

  Run:

  ```powershell
  python -m pytest docreader/tests/test_excel_parser.py docreader/tests/test_docx_merge.py docreader/tests/test_docx_tables.py docreader/tests/test_pdf_embedded_images.py docreader/tests/test_tempfile_context.py -q
  ```

- [ ] **Step 3: Adapt the official implementation file by file.**

  Preserve Xelora's lightweight `Document` model and inline `ImageRef` flow. Do not import official storage, migration, or engine-registry code. For subprocess fixes, terminate the process group and remove only the task's temporary directory.

- [ ] **Step 4: Run the focused and package tests.**

  Run:

  ```powershell
  python -m pytest docreader/tests/test_excel_parser.py docreader/tests/test_docx_merge.py docreader/tests/test_docx_tables.py docreader/tests/test_pdf_embedded_images.py docreader/tests/test_tempfile_context.py docreader/tests/test_markdown_table_util.py -q
  python -m compileall -q docreader
  git diff --check
  ```

- [ ] **Step 5: Commit the P0 batch.**

  ```powershell
  git add docreader/parser docreader/utils docreader/tests
  git commit -m "fix: backport safe document parser correctness fixes"
  ```

  Rollback point: revert this commit as a unit if any existing parser regression appears; `xlsx_visual` work must be based on a passing P0 batch.

### Task 3: Add the `xlsx_visual` provider contract and registry entry

**Files:**
- Create: `docreader/parser/xlsx_visual_parser.py`
- Create: `docreader/parser/xlsx_manifest.py`
- Modify: `docreader/parser/registry.py`
- Test: `docreader/tests/test_xlsx_visual_parser.py`
- Test: `docreader/tests/test_config.py`

**Interfaces:**
- Consumes: `BaseParser.parse(content: bytes) -> Document`, `parser_engine_overrides`, and the existing `Document(content, images, metadata)` model.
- Produces: `XlsxVisualParser.parse_into_text(content: bytes) -> Document`, registered under `xlsx_visual` for `xlsx` and `xls`, with override keys `xlsx_sheet_mode`, `xlsx_canvas_mode`, `xlsx_render_backend`, `xlsx_include_drawing_metadata`, `xlsx_preserve_sheet_images`, and `xlsx_visual_vlm`.

- [ ] **Step 1: Write routing and contract tests.**

  Assert that `registry.get_parser_class("xlsx_visual", "xlsx")` returns `XlsxVisualParser`, `builtin` still returns `ExcelParser`, unknown extensions still fall back as before, and the parser returns non-empty Markdown plus deterministic metadata for a table-only fixture.

- [ ] **Step 2: Run the tests to establish the missing provider failure.**

  Run `python -m pytest docreader/tests/test_xlsx_visual_parser.py -q`.

- [ ] **Step 3: Implement the smallest provider facade.**

  Define a `XlsxVisualParser` constructor accepting the existing `file_name`, `file_type`, and keyword overrides. Its first path loads the workbook, delegates `table` sheets to the existing table conversion without changing `ExcelParser`, emits `index.md`-equivalent content, and writes a deterministic `manifest.json` entry into `Document.metadata`; it must not render or call a model yet.

- [ ] **Step 4: Register the provider and validate engine listing.**

  Add one `registry.register("xlsx_visual", {"xlsx": XlsxVisualParser, "xls": XlsxVisualParser}, ...)` entry. Keep `builtin` registration unchanged and confirm `ListEngines` exposes both names.

- [ ] **Step 5: Run tests and commit the provider skeleton.**

  ```powershell
  python -m pytest docreader/tests/test_xlsx_visual_parser.py docreader/tests/test_excel_parser.py docreader/tests/test_config.py -q
  git add docreader/parser/xlsx_visual_parser.py docreader/parser/xlsx_manifest.py docreader/parser/registry.py docreader/tests
  git commit -m "feat: register xlsx visual parser provider"
  ```

  Rollback point: remove only the provider registration and new modules; the `builtin` path remains deployable.

### Task 4: Implement worksheet classification and DrawingML metadata

**Files:**
- Create: `docreader/parser/xlsx_classifier.py`
- Create: `docreader/parser/xlsx_drawing.py`
- Modify: `docreader/parser/xlsx_visual_parser.py`
- Modify: `docreader/parser/xlsx_manifest.py`
- Test: `docreader/tests/test_xlsx_classifier.py`
- Test: `docreader/tests/test_xlsx_drawing.py`
- Test fixtures: `docreader/tests/fixtures/xlsx_visual/` containing dense table, merged table, native chart, image-only, flowchart, mixed, hidden-sheet, and multilingual workbooks.

**Interfaces:**
- Consumes: original workbook bytes, `openpyxl` workbook objects, ZIP/XML parts under `xl/drawings/`, chart relationships, worksheet visibility, cell ranges, and drawing anchors.
- Produces: `classify_workbook(content: bytes) -> list[SheetClassification]` and `extract_drawing_metadata(content: bytes, sheet_index: int) -> list[DrawingObject]`; classifications are exactly `table`, `native_chart`, `canvas`, `mixed`, or `image_only`.

- [ ] **Step 1: Create fixture builders and failing classification tests.**

  Each fixture must assert evidence fields: non-empty cell count, used-range density, formula count, chart count, image count, shape/connector count, text-bearing shape count, and estimated drawing area. Borderline evidence must resolve to `mixed`.

- [ ] **Step 2: Implement evidence-based classification.**

  Inspect visible and relevant hidden sheets before merge normalization. Use explicit evidence thresholds in one classifier module; do not classify from a single cell-density heuristic. Keep native charts separate from canvas sheets when supporting data exists.

- [ ] **Step 3: Implement DrawingML extraction.**

  Read worksheet relationships and drawing XML with `zipfile` and `lxml`/stdlib XML. Return stable object IDs, object type, text, cell anchors, bounding boxes when available, relationship IDs, and chart/image references. Malformed drawing XML becomes a warning and does not discard the sheet.

- [ ] **Step 4: Add manifest serialization tests.**

  Assert stable JSON ordering, source SHA-256, sheet index/name/visibility/classification/evidence, object IDs, and warning records. The same fixture bytes must produce the same manifest bytes across two runs.

- [ ] **Step 5: Run and commit.**

  ```powershell
  python -m pytest docreader/tests/test_xlsx_classifier.py docreader/tests/test_xlsx_drawing.py docreader/tests/test_xlsx_visual_parser.py -q
  git add docreader/parser docreader/tests/fixtures/xlsx_visual docreader/tests/test_xlsx_*.py
  git commit -m "feat: classify xlsx sheets and extract drawing metadata"
  ```

  Rollback point: revert classifier/drawing modules without changing provider registration or ordinary Excel parsing.

### Task 5: Add deterministic worksheet rendering and visual output assembly

**Files:**
- Create: `docreader/parser/xlsx_renderer.py`
- Modify: `docreader/parser/xlsx_visual_parser.py`
- Modify: `docreader/parser/xlsx_manifest.py`
- Test: `docreader/tests/test_xlsx_renderer.py`
- Test: `docreader/tests/test_xlsx_visual_parser.py`

**Interfaces:**
- Consumes: `SheetClassification`, original workbook bytes, renderer override `xlsx_render_backend`, and timeout settings.
- Produces: `render_sheet(content: bytes, sheet_index: int, backend: str, timeout_seconds: float) -> RenderedSheet` containing PNG/PDF bytes, dimensions, backend, and SHA-256; `Document.images` maps deterministic asset paths to base64 payloads.

- [ ] **Step 1: Add backend and failure tests.**

  Cover backend selection order `excel_com -> libreoffice -> explicit failure`, missing executables, timeout cleanup, sheet-specific output, stable asset names, and a single bad sheet leaving other sheets available.

- [ ] **Step 2: Implement isolated rendering.**

  Use an isolated temporary directory per parse. Excel COM is attempted only when the Windows automation dependency is importable; LibreOffice uses a headless subprocess with a bounded timeout. On timeout, terminate the process tree and record request/sheet/backend/elapsed-time diagnostics.

- [ ] **Step 3: Assemble Markdown and image references.**

  `table` sheets use structured text only; `native_chart`, `canvas`, `mixed`, and `image_only` sheets include a Markdown image reference and concise structured metadata. Use `ReadStream` for large image sets rather than changing the Go image contract.

- [ ] **Step 4: Verify rendering-unavailable and multilingual fixtures.**

  Run `python -m pytest docreader/tests/test_xlsx_renderer.py docreader/tests/test_xlsx_visual_parser.py -q`; assert a warning/error is present and the provider never returns an empty successful document.

- [ ] **Step 5: Commit the rendering path.**

  ```powershell
  git add docreader/parser/xlsx_renderer.py docreader/parser/xlsx_visual_parser.py docreader/parser/xlsx_manifest.py docreader/tests
  git commit -m "feat: render visual xlsx sheets with deterministic assets"
  ```

  Rollback point: revert the renderer commit; `xlsx_visual` remains usable for table-only workbooks through the provider skeleton.

### Task 6: Extend the gRPC contract and Go document-reader mapping

**Files:**
- Modify: `docreader/proto/docreader.proto`
- Regenerate: `docreader/proto/docreader_pb2.py`, `docreader/proto/docreader_pb2.pyi`, `docreader/proto/docreader_pb2_grpc.py`, `docreader/proto/docreader.pb.go`, `docreader/proto/docreader_grpc.pb.go`
- Modify: `docreader/main.py`
- Modify: `internal/types/docparser.go`
- Modify: `internal/infrastructure/docparser/grpc_parser.go`
- Test: `docreader/client/client_test.go`
- Test: `internal/infrastructure/docparser/grpc_parser_test.go`

**Interfaces:**
- Consumes: provider manifest/source locators and current `Read`/`ReadStream` frames.
- Produces: `SourceBlock` with `block_id`, `sheet_name`, `sheet_index`, `block_type`, `object_id`, `asset_ref`, and `locator_json`; `ReadResponse.source_blocks` and `ReadStreamMeta.source_blocks` use appended field numbers only. Go maps these to `types.SourceBlock` without changing existing `ImageRef` behavior.

- [ ] **Step 1: Add compatibility tests before changing the schema.**

  Assert that old responses without source blocks still decode, that new responses preserve Markdown/images, and that `ReadStream` emits source blocks only in the metadata frame.

- [ ] **Step 2: Append the protobuf messages and fields.**

  Add `message SourceBlock` and append `repeated SourceBlock source_blocks` to `ReadResponse` and `ReadStreamMeta` with new field numbers. Never renumber or remove fields 1-5 already in use.

- [ ] **Step 3: Regenerate both language stubs with the repository script.**

  Run `powershell -NoProfile -Command "./docreader/scripts/generate_proto.sh"` from the repository root, then inspect the generated diff to ensure only the intended proto outputs changed.

- [ ] **Step 4: Map metadata in Python and Go.**

  `docreader/main.py` converts provider source blocks to UTF-8-safe protobuf values. `internal/types/docparser.go` adds `SourceBlock`; `grpc_parser.go` copies blocks in both unary and streaming paths. Existing callers that ignore blocks continue to work.

- [ ] **Step 5: Run protocol tests and commit.**

  ```powershell
  python -m pytest docreader/tests -q
  go test ./internal/infrastructure/docparser ./internal/types/...
  git diff --check
  git add docreader/proto docreader/main.py internal/types/docparser.go internal/infrastructure/docparser/grpc_parser.go docreader/client
  git commit -m "feat: expose xlsx source locators through docreader protocol"
  ```

  Rollback point: revert the schema and generated stubs together; do not deploy a Python stub set that differs from the Go stubs.

### Task 7: Connect the provider output to knowledge ingestion and model-independent VLM fallback

**Files:**
- Modify: `internal/infrastructure/docparser/image_resolver.go`
- Modify: `internal/types/docparser.go`
- Modify: `internal/application/service/knowledge_process.go`
- Modify: `internal/handler/session/attachment_processor.go`
- Test: `internal/infrastructure/docparser/image_resolver_test.go`
- Test: `internal/infrastructure/docparser/markdown_image_scanner_test.go`
- Test: `internal/application/service/knowledge_process_test.go`
- Test: `internal/handler/session/attachment_processor_test.go`

**Interfaces:**
- Consumes: `ReadResult.MarkdownContent`, `ReadResult.ImageRefs`, and optional `SourceBlocks`.
- Produces: unchanged image persistence/OCR/VLM/chunk behavior, with source locator metadata attached to parsed chunks where the existing metadata path supports it.

- [ ] **Step 1: Add an end-to-end fixture test for image reference resolution.**

  Feed the generated Markdown and `ImageRef` list into the current Go resolver; assert every visual asset is stored once, referenced images resolve, and table-only output behaves exactly as before.

- [ ] **Step 2: Preserve failure semantics.**

  A failed image upload remains a Go-side parse failure/retry according to existing status handling. A failed VLM call preserves the rendered image and DrawingML metadata; it must not erase the source block or mark the whole workbook empty.

- [ ] **Step 3: Validate direct and agent-mediated retrieval.**

  Use an isolated test knowledge base with the flowchart, mixed, and table fixtures. Verify direct knowledge-base queries and agent-mediated knowledge-base queries both retrieve the same visual-sheet evidence and do not require a model call for ordinary tables.

- [ ] **Step 4: Run focused Go tests and commit.**

  ```powershell
  go test ./internal/infrastructure/docparser ./internal/application/service/... -run 'Test.*(Image|Document|Chunk)' -count=1
  git add internal/infrastructure/docparser internal/types internal/application
  git commit -m "feat: preserve xlsx visual assets through knowledge ingestion"
  ```

  Rollback point: revert only ingestion metadata changes; image storage and existing OCR/VLM code remain unchanged.

### Task 8: Add acceptance coverage, operational documentation, and release checks

**Files:**
- Create: `docreader/tests/test_xlsx_visual_acceptance.py`
- Create: `docs/operations/xlsx-visual-parser.md`
- Modify: `docreader/README.md`
- Modify: `docs/superpowers/specs/2026-10-12-xlsx-visual-parser-design.md` only to record implemented deviations
- Test: all `docreader/tests/test_xlsx_*.py` and affected Go packages

**Interfaces:**
- Consumes: all provider, protocol, and ingestion behavior from Tasks 2-7.
- Produces: a release checklist that can prove source/asset hashes, stream behavior, fallback diagnostics, ordinary table compatibility, and deployment readiness.

- [ ] **Step 1: Add the complete fixture matrix.**

  Acceptance tests must cover dense table, merged table, native chart, image-only sheet, flowchart with arrows/text boxes, mixed sheet, hidden sheet, multilingual sheet names/text, unavailable renderer, unavailable/timed-out VLM, and a large workbook requiring `ReadStream`.

- [ ] **Step 2: Document configuration and rollback.**

  Document `parser_engine: "xlsx_visual"`, every override key, backend prerequisites, warning semantics, image-size limits, and the rollback procedure of selecting `builtin` for ordinary Excel uploads.

- [ ] **Step 3: Run the complete verification set.**

  ```powershell
  python -m pytest docreader/tests -q
  go test ./internal/infrastructure/docparser ./internal/application/service ./internal/middleware -count=1
  go test ./... -run 'Test.*Doc(Parser|Reader|Image|Chunk)' -count=1
  git diff --check
  ```

- [ ] **Step 4: Review the implementation against the spec.**

  Confirm every spec section has a passing test or an explicit documented limitation, no migration files changed, no default engine behavior changed, and no generated protobuf file is stale.

- [ ] **Step 5: Commit release documentation and tag the implementation checkpoint.**

  ```powershell
  git add docreader/tests/test_xlsx_visual_acceptance.py docs/operations/xlsx-visual-parser.md docreader/README.md docs/superpowers/specs/2026-10-12-xlsx-visual-parser-design.md
  git commit -m "docs: document xlsx visual parser operations and acceptance"
  git push origin feat/xlsx-visual-parser
  ```

  Rollback point: use the last passing task commit and select `builtin` for Excel uploads while visual parsing is disabled.

## Verification and Integration Notes

- Do not merge `tencent-v0.8.2` wholesale: the Xelora branch has no common merge base with that official tag, and Xelora migration numbers and engine contracts differ.
- Keep all commits on `feat/xlsx-visual-parser` until the full acceptance matrix passes. The branch is already published as `origin/feat/xlsx-visual-parser`.
- Before any runtime deployment, build only the affected DocReader/App images, verify generated protobuf hashes in both Python and Go trees, and keep the previous image tag available for rollback.
- No implementation code is included in this plan commit; execution begins only after the plan is reviewed and selected as the next work mode.
