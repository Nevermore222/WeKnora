---
name: workspace-file-writer
description: Use this skill when the user needs a real markdown, text, JSON, or CSV file created or updated in the workspace instead of only receiving text in chat. It wraps safe relative-path file operations behind a structured JSON request.
---

# Workspace File Writer

Use this skill when the primary output should be a real file such as:

- `.md`
- `.txt`
- `.json`
- `.csv`

Use `officecli-document-editing` instead when the user needs `.docx`, `.xlsx`,
or `.pptx`.

## How to run it

1. Call `execute_skill_script`
2. Use `skill_name="workspace-file-writer"`
3. Use `script_path="scripts/workspace_file_writer.py"`
4. Pass a relative JSON request filename as the first positional arg, for
   example `["request.json"]`
5. Put the request JSON content into `input`
6. If the model accidentally passes the request object directly as a JSON
   string argument, the script also accepts that form as a fallback

## Request format

The request JSON must be an object.

### Write or overwrite a markdown file

```json
{
  "action": "write",
  "file": "reports/project-overview.md",
  "content": "# Project Overview\n\nThis file was created by Xelora.\n",
  "overwrite": true
}
```

### Append to an existing markdown file

```json
{
  "action": "append",
  "file": "reports/project-overview.md",
  "content": "\n## Next Steps\n\n- Validate runtime\n- Ship preview\n"
}
```

### Write JSON with stable formatting

```json
{
  "action": "write_json",
  "file": "exports/summary.json",
  "data": {
    "project": "Xelora",
    "status": "active"
  },
  "indent": 2
}
```

### Copy an existing file

```json
{
  "action": "copy",
  "source_file": "reports/project-overview.md",
  "file": "reports/project-overview-copy.md",
  "overwrite": true
}
```

## Rules

- All relative paths resolve from the current conversation's bound workspace.
- If the conversation has no valid workspace binding, the executor returns
  `workspace_required` before this script runs.
- Always keep paths relative. Never use absolute paths.
- Prefer subdirectories like `reports/`, `exports/`, or `artifacts/` when the
  file is part of a larger workflow.
- If the user asks for a real markdown file, do not stop at writing markdown in
  chat. Run the script and create the artifact.
- Use `write_json` for JSON output instead of building raw JSON text by hand.
