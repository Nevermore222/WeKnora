# Xelora Excel Visual Parser Design

**Date:** 2026-10-12

## Decision

Extend the existing Xelora/WeKnora document-processing pipeline with a
selectable `xlsx_visual` parser provider. Reuse the existing document-reader
protocol, Go-side image storage, OCR/VLM processing, chunking, embedding,
reference chain, task queues, and knowledge-base lifecycle. Do not create a second
knowledge-ingestion system and do not replace the existing builtin Excel
parser for ordinary tabular workbooks.

The first implementation has two independent tracks:

1. Selectively backport low-conflict parser fixes from official WeKnora
   `tencent-v0.8.2`.
2. Add Xelora-owned workbook inspection, worksheet classification, rendering,
   visual transcription, and output assembly behind `xlsx_visual`.

The provider returns the same logical document result as the existing parser:
Markdown text plus image references. The Go application remains the owner of
image persistence, OCR/caption tasks, chunking, indexing, and retrieval.

## Repository Baseline

- Xelora repository: `E:/Xelora/WeKnora`
- Active branch: `knavis-internal`
- Current Xelora commit: `5b1c17f5`
- Xelora secondary-development baseline: `06b9cc36`
- Official comparison tag: `tencent-v0.8.2` (`3e8b0bfc`)

The histories have no common merge base. Official changes must therefore be
reviewed and adapted file by file; the official branch must not be merged as a
whole.

## Goals

- Preserve ordinary `builtin` Excel parsing for table-oriented sheets.
- Detect workbook sheet roles before mutating or normalizing workbook bytes.
- Preserve white-canvas sheets containing shapes, text boxes, arrows,
  connectors, images, and charts.
- Produce searchable Markdown while retaining rendered visual evidence.
- Reuse existing Go image storage and multimodal processing.
- Make every sheet and asset traceable to the source workbook and parser run.
- Allow per-upload overrides through the existing parser override contract.
- Keep old `Read` and `ReadStream` consumers working.

## Non-goals

- Do not replace the existing chunker, image resolver, OCR/VLM tasks, or
  knowledge post-processing.
- Do not make `ExcelParser` understand absolute DrawingML layout.
- Do not make the Python parser persist objects directly to MinIO, COS, or
  local storage.
- Do not require all worksheets to be converted into one flat table.
- Do not copy official database migrations directly; Xelora migration numbers
  `000064` through `000070` already have different meanings.
- Do not make a model call mandatory for ordinary table sheets.

## Existing Xelora Capabilities To Preserve

The current tree already provides:

- `Read` and `ReadStream` gRPC calls;
- `ImageRef` with `image_data` fallback;
- Go-side image resolution and storage;
- OCR/VLM, image caption, and image OCR child chunks;
- parser-engine selection and `parser_engine_overrides`;
- builtin, MarkItDown, OpenDataLoader, MinerU, PaddleOCR-VL, and XeloraCloud
  paths;
- asynchronous knowledge post-processing, chunking, embedding, and retrieval;
- DuckDB-backed Excel data analysis.

These contracts remain authoritative. `xlsx_visual` is an additional provider,
not a replacement for those subsystems.

## Official Backport Matrix

### Priority P0: correctness and data-loss fixes

- Excel image-function filtering (`DISPIMG`/`IMAGE`).
- XLS/XLSX first-row header override behavior.
- XLSX merge preprocessing that removes charts before temporary save.
- DOCX vertical-merge preservation and DOCX table retention.
- Legacy binary DOC detection when content is mislabeled as DOCX.
- PDF embedded-image SMask handling and pdfium bitmap lifetime safety.
- Converter timeout process-group termination.
- Temporary-file cleanup on failed context entry.

### Priority P1: routing and format coverage

- HTML/HTM local upload parser and registration.
- XMind parser and registration.
- PPT/PPTX default MarkItDown routing.
- EPUB/MHTML registration and header decoding fixes.
- OpenDataLoader availability probes and redirect/SSRF checks.

### Priority P1: security and resilience

- SSRF checks before DNS and redirect blocking.
- Markdown-table catastrophic-backtracking fix.
- Parser routing, sandbox, SSRF, concurrency, and temporary-file tests.

Every backport must be adapted to Xelora package names, engine registry
behavior, process configuration, and existing image/post-processing tests.

## `xlsx_visual` Components

The provider is split into small units with stable interfaces:

```text
xlsx_visual_parser.py       provider facade and orchestration
xlsx_classifier.py          workbook/sheet role classification
xlsx_renderer.py            Excel COM or LibreOffice rendering
xlsx_drawing.py             DrawingML metadata and object relationships
xlsx_manifest.py            deterministic manifest and warnings
```

The exact module names may change during implementation, but the ownership
boundaries must remain.

### Classification

Each visible and relevant hidden worksheet is classified independently as:

- `table`: dense cell values, no meaningful drawing layer;
- `native_chart`: native Excel chart object with supporting cell data;
- `canvas`: sparse cells plus shapes, text boxes, connectors, or large images;
- `mixed`: meaningful table content and a meaningful drawing layer;
- `image_only`: visual content with negligible cell text.

Classification must record evidence rather than rely on a single threshold:
used-range density, non-empty cell count, formulas, chart objects, image count,
shape count, connector count, text-bearing shape count, and estimated drawing
area. Borderline sheets are classified as `mixed` and retain both paths.

### Processing paths

`table`:

```text
existing ExcelParser or configured fallback
  -> Markdown/table content
  -> Go chunking and indexing
```

`native_chart`:

```text
structured cell/chart metadata
  + rendered chart/sheet image
  -> Markdown with image reference
  -> existing Go image and text pipeline
```

`canvas`, `mixed`, and `image_only`:

```text
preserve original workbook bytes
  -> render sheet to PNG/PDF
  -> extract DrawingML object metadata
  -> optional VLM transcription
  -> Markdown image reference and structured locator metadata
  -> existing Go storage/OCR/VLM/chunk pipeline
```

Merged-cell normalization is applied only to the table path. The visual path
must render and inspect the original workbook first.

## Rendering

The renderer is provider-configured and reports the backend in the manifest.

Preferred order on Windows:

1. Excel COM when the required desktop automation environment is available;
2. LibreOffice headless conversion as a deterministic fallback;
3. explicit failure with a diagnostic warning when neither backend is usable.

Rendering must use isolated temporary directories, bounded timeouts, and
process-tree cleanup. It must never mutate the uploaded source workbook.

For each rendered sheet, retain page dimensions, DPI/backend, sheet index,
sheet name, and an SHA-256 hash of the generated asset.

## Protocol Extension

The current Xelora `docreader.proto` contains `Read`, `ReadStream`, `ImageRef`,
and metadata but no `SourceBlock`. Add an optional backward-compatible source
locator extension rather than changing the meaning of existing fields.

Proposed fields:

```text
message SourceBlock {
  string block_id = 1;
  string sheet_name = 2;
  uint32 sheet_index = 3;
  string block_type = 4;       // table, chart, shape, image, rendered_page
  string object_id = 5;
  string asset_ref = 6;
  string locator_json = 7;     // bbox, cell range, relationship metadata
}
```

Add `repeated SourceBlock source_blocks` to both `ReadResponse` and
`ReadStreamMeta` using new field numbers. Do not remove or renumber existing
fields. Generated Go and Python stubs must be regenerated together. Older
consumers continue to use Markdown and `ImageRef` only.

The first implementation may keep locator data in `metadata` if the protocol
extension is deferred, but the manifest remains the authoritative detailed
record until the typed field is available.

## Output Contract

The provider assembles a deterministic logical document package:

```text
index.md
sheets/01-sheet.md
sheets/02-sheet.md
assets/sheet-01-page-01.png
assets/sheet-01.pdf
manifest.json
warnings.json
source.xlsx.sha256
```

`index.md` links every sheet and asset. Sheet Markdown includes the original
sheet name, classification, structured text when available, image references,
and concise VLM transcription when enabled. Visual assertions must cite the
rendered asset rather than pretending that Markdown preserves coordinates.

The manifest records:

- source hash and file size;
- parser/provider version;
- sheet index, name, visibility, classification, and evidence;
- cell ranges, drawing object IDs, chart metadata, and bounding boxes;
- render backend and asset hashes;
- VLM/OCR status and model-independent warnings;
- fallback path and error diagnostics.

## Configuration

Reuse existing `parser_engine_overrides`. Initial keys:

```json
{
  "xlsx_sheet_mode": "auto",
  "xlsx_canvas_mode": "render",
  "xlsx_render_backend": "auto",
  "xlsx_include_drawing_metadata": "true",
  "xlsx_preserve_sheet_images": "true",
  "xlsx_visual_vlm": "auto"
}
```

The explicit engine name is:

```json
{ "parser_engine": "xlsx_visual" }
```

`auto` classification must be the default for `xlsx_visual`. Ordinary
`builtin` behavior remains unchanged until a separate product decision enables
automatic routing to `xlsx_visual`.

## Error Handling

- A failed sheet classification falls back to `mixed` and preserves a rendered
  sheet image when possible.
- A failed renderer returns a structured parser error; it must not silently
  return an empty document.
- A failed VLM call preserves the rendered image and structured DrawingML
  metadata, and records the failure in `warnings.json`.
- A single bad sheet must not discard successfully processed sheets.
- Image upload/storage failures remain owned by the existing Go resolver and
  must follow its retry and parse-status behavior.
- Provider timeouts terminate child processes and include request ID, sheet,
  backend, and elapsed time in diagnostics.

## Verification Plan

### Backport verification

- Focused parser tests for every P0 backport.
- Existing Xelora docreader and Go docparser tests.
- No migration changes as part of parser backport work.
- `git diff --check` and package-level formatting/type checks.

### `xlsx_visual` verification

Fixtures must cover:

- dense ordinary table;
- merged table;
- native chart with supporting data;
- image-only sheet;
- flowchart with arrows and text boxes;
- mixed table plus drawing layer;
- hidden sheet;
- multilingual sheet names and cell text;
- rendering unavailable;
- VLM unavailable or timed out;
- large workbook requiring `ReadStream`.

Acceptance requires:

- all expected sheets represented;
- no table sheet silently dropped;
- every visual sheet has a valid asset or explicit failure warning;
- source hash and asset hashes are stable;
- Markdown references resolve to returned `ImageRef` entries;
- Go-side image storage and OCR/VLM tasks receive the same image contract as
  existing PDF/DOCX parsing;
- ordinary table parsing remains behaviorally compatible.

## Delivery Phases

1. **Audit and P0 backports:** implement only official correctness fixes and
   their focused tests.
2. **Provider skeleton:** register `xlsx_visual`, define configuration and
   deterministic manifest, and return table-path results first.
3. **Classification and rendering:** add canvas/mixed/image-only handling with
   asset streaming and failure diagnostics.
4. **Drawing metadata and VLM:** add object relationships, locators, and
   optional transcription without changing Go chunk ownership.
5. **Knowledge-base integration:** connect the provider output to the existing
   ingestion path, validate direct KB queries and agent-mediated KB queries,
   and update operational documentation.

Implementation must not begin until this specification is reviewed and
approved. After approval, create a separate implementation plan with explicit
file lists, test commands, and rollback points.
