---
name: officecli-document-editing
description: Use this skill when the primary deliverable is a real .docx, .xlsx, or .pptx file that must be created, inspected, or modified in the workspace. It wraps OfficeCLI through a structured JSON request file so web agents can reliably produce Office artifacts instead of only text descriptions.
---

# OfficeCLI Document Editing

This skill is the first Xelora-native bridge for real Office file work inside
the controlled Docker sandbox.

Use it when the user needs:

- a new `.docx`, `.xlsx`, or `.pptx` file
- changes to an existing Office file
- document structure inspection before editing
- text, html, screenshot, or pdf preview of an Office file
- OpenXML validation after editing

Do not use it when the primary deliverable is Markdown, HTML, a Python script,
or a database result.

## How to run it

1. Call `execute_skill_script`
2. Use `skill_name="officecli-document-editing"`
3. Use `script_path="scripts/officecli_bridge.py"`
4. Pass a relative JSON request filename as the first positional arg, for
   example `["request.json"]`
5. Put the request JSON content into `input`

The backend will materialize the JSON file in the skill workspace before the
script runs.

## Request format

The request JSON must be an object.

Ready-to-reuse request examples live under `examples/`.

### Create a blank file

```json
{
  "action": "create",
  "file": "quarterly-review.pptx",
  "force": true
}
```

### Write a Word document from generated text

Use this for generated prose, articles, or long-form text that should become a
real `.docx` file. It creates the document, adds the title and paragraphs,
validates the result, and atomically replaces the target file only after
validation succeeds.

```json
{
  "action": "write_docx",
  "file": "sanzijing.docx",
  "title": "三字经",
  "paragraphs": [
    "人之初，性本善。",
    "苟不教，性乃迁。"
  ],
  "force": true
}
```

You can also pass newline-separated text as `"content"` instead of
`"paragraphs"` when that is more compact.

### Write an Excel workbook from structured rows

Use this for generated tables, summaries, or knowledge-base exports that should
become a real `.xlsx` file. It creates the workbook in one script call,
validates that the workbook can be opened, and atomically replaces the target
file only after validation succeeds.

```json
{
  "action": "write_xlsx",
  "file": "knowledge-summary.xlsx",
  "sheets": [
    {
      "name": "Summary",
      "headers": ["Topic", "Status"],
      "rows": [
        ["Chrome DevTools MCP", "Ready"]
      ]
    }
  ],
  "force": true
}
```

You can also pass top-level `"sheet"`, `"headers"`, and `"rows"` when the
workbook only needs one sheet.

### Validate an Office file

```json
{
  "action": "validate",
  "file": "quarterly-review.pptx"
}
```

### Read a DOM node

```json
{
  "action": "get",
  "file": "brief.docx",
  "path": "/body/p[1]",
  "depth": 2
}
```

### Query elements

```json
{
  "action": "query",
  "file": "brief.docx",
  "selector": "paragraph"
}
```

### Modify a node

```json
{
  "action": "set",
  "file": "brief.docx",
  "path": "/body/p[1]",
  "props": {
    "text": "Updated title"
  }
}
```

### Add a node

```json
{
  "action": "add",
  "file": "deck.pptx",
  "parent": "/slide[1]",
  "type": "shape",
  "props": {
    "text": "Executive summary",
    "x": "1cm",
    "y": "1cm",
    "width": "8cm"
  }
}
```

### Batch edit

```json
{
  "action": "batch",
  "file": "deck.pptx",
  "commands": [
    {
      "command": "add",
      "parent": "/",
      "type": "slide"
    },
    {
      "command": "add",
      "parent": "/slide[1]",
      "type": "shape",
      "props": {
        "text": "Hello Xelora"
      }
    }
  ]
}
```

### Preview a file

```json
{
  "action": "view",
  "file": "brief.docx",
  "mode": "text",
  "max_lines": 20
}
```

For `view`, the most useful modes are:

- `text`: fast text preview for docx, xlsx, or pptx
- `html`: browser-ready preview, usually with `"out": "preview.html"`
- `screenshot`: image preview, usually with `"out": "preview.png"`
- `pdf`: exported PDF, usually with `"out": "preview.pdf"`

## Rules

- All document and preview paths resolve from the current conversation's bound
  workspace, not from this skill package.
- If the conversation has no valid workspace binding, the executor returns
  `workspace_required` before this script runs.
- Always keep file paths relative. Never pass absolute paths.
- Run `validate` after meaningful edits unless the user explicitly says not to.
- Prefer `write_docx` for generated long-form Word documents because it keeps
  tool-call payloads compact and avoids partial batch requests.
- Prefer `write_xlsx` for generated spreadsheets because it avoids creating an
  empty workbook and editing cells one by one. Use lower-level `batch` only when
  updating a specific part of an existing workbook.
- Prefer `batch` for multi-step document changes because it is more stable than
  many single commands.
- Prefer the `examples/` request shapes when building the first draft of a new
  Office task.
- If the user wants a real file, do not stop at describing the file. Run the
  script and produce the artifact.
