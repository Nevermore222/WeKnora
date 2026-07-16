# Xelora n8n Parameter Parse Migration Design

## Goal

Create a new parallel Xelora-backed n8n workflow for CL command parameter parsing, using the existing Dify/RAGFlow-backed workflow as the reference implementation.

The first migration scope is only command parameter parsing. The existing workflow must remain unchanged and runnable during the early migration phase. n8n remains responsible for workflow orchestration, database reads and writes, retry handling, response validation, and failure recording. Xelora provides the dedicated agent, the `Manual_ASP` knowledge base, streaming answer generation, and strict JSON output.

## Current Context

The existing n8n parameter parsing workflow receives command context, reads `source_content` from `analyzes.command_reference_list`, merges those snippets, calls a Dify agent, parses the streamed response, and stores parsed parameter data.

The new parallel workflow keeps the high-level n8n shape but removes the database `SOURCE` payload from the LLM request. The Xelora agent must use the `Manual_ASP` knowledge base as the factual source for command parameter extraction.

The Dify/RAGFlow workflow is not replaced in this phase. It remains the baseline path for production comparison and rollback.

## Decisions

- First new parallel workflow: command parameter parsing.
- Xelora agent: one dedicated CL command parameter parsing agent.
- Knowledge base: existing `Manual_ASP`.
- Authentication from n8n to Xelora: `X-API-Key`.
- Session policy: create a new Xelora session for every workflow invocation.
- Output protocol: strict JSON, aligned with `command_parameters`.
- Retry policy: retry strict JSON once. If the second attempt fails, record failure and do not write parameter rows.
- Skill usage: no skill for the first parameter parsing phase. Later verification-case table generation and TOIN&FS code generation should use dedicated skills.
- Migration policy: add a new n8n workflow instead of editing or replacing the existing Dify/RAGFlow workflow.
- Suggested workflow name: `Xelora - CL Parameter Parse - Parallel`.
- Suggested execution mode: manual or isolated webhook during validation, not the existing production trigger.

## n8n Workflow

The new parallel n8n workflow receives:

```json
{
  "command_id": 123,
  "command": "SORTD",
  "language": "CL"
}
```

The workflow steps are:

1. Receive `command_id`, `command`, and `language` from webhook input.
2. Create a new Xelora session with `POST /api/v1/sessions`.
3. Call `POST /api/v1/agent-chat/{session_id}` with the dedicated agent and `Manual_ASP`.
4. Consume the SSE response and concatenate `answer` chunks until `complete`.
5. Parse the final answer as JSON.
6. Validate the JSON shape.
7. Normalize field values for `command_parameters`.
8. Insert or update parameter records.
9. If JSON parsing or validation fails, retry once with a new session.
10. If the retry fails, record a workflow failure with the raw response.

The workflow must not query or pass `analyzes.command_reference_list.source_content` to Xelora in this phase.

The workflow should be copied or recreated from the existing n8n parameter parsing workflow only as a structural reference. It must not overwrite the original workflow ID, original webhook path, original credentials, or original production schedule.

During validation, writes should be isolated from the original workflow when possible:

- Preferred: write to a comparison table or staging table for Xelora parameter parsing results.
- Acceptable: write to `command_parameters` only when an explicit `source_system = xelora` or equivalent isolation field exists.
- Not acceptable: silently overwrite rows produced by the existing Dify/RAGFlow workflow.

## Xelora Request

n8n creates a session:

```http
POST /api/v1/sessions
X-API-Key: {N8N_XELORA_API_KEY}
Content-Type: application/json
```

```json
{
  "title": "CL parameter parse SORTD"
}
```

n8n then calls the agent:

```http
POST /api/v1/agent-chat/{session_id}
X-API-Key: {N8N_XELORA_API_KEY}
Content-Type: application/json
```

```json
{
  "query": "Parse all parameters of CL command SORTD. Return only one JSON object that matches the agreed schema. Do not return Markdown or explanatory text.",
  "agent_enabled": true,
  "agent_id": "{CL_PARAMETER_AGENT_ID}",
  "knowledge_base_ids": ["{MANUAL_ASP_KB_ID}"],
  "mentioned_items": [
    {
      "id": "{MANUAL_ASP_KB_ID}",
      "name": "Manual_ASP",
      "type": "kb",
      "kb_type": "document"
    }
  ],
  "web_search_enabled": false,
  "channel": "api"
}
```

The query should include the command name and language. It should not include database source text.

## Agent Prompt Boundary

The dedicated Xelora agent is responsible only for CL command parameter parsing.

The system prompt must require:

- Use `Manual_ASP` as the factual source.
- Parse only the requested command.
- Return exactly one JSON object.
- Do not output Markdown fences, explanatory prose, or comments.
- Do not fabricate parameters when the knowledge base lacks evidence.
- Keep field names exactly aligned with the schema.
- Use empty strings for unknown optional string fields.
- Use `false` for unknown `is_required`.
- Keep each logical command parameter as one object.

The agent must not generate verification cases, project filtering results, CL code, COBOL code, or PF/SF files in this phase.

## JSON Schema

The agent output must have this shape:

```json
{
  "command": "SORTD",
  "language": "CL",
  "parameters": [
    {
      "parameter_name": "PARAM",
      "parameter_type": "SIMPLE",
      "data_type": "STRING",
      "enum_value": "",
      "value_range": "",
      "default_value": "",
      "description": "",
      "is_required": false,
      "relationship_notes": ""
    }
  ]
}
```

Required root fields:

- `command`
- `language`
- `parameters`

Required parameter fields:

- `parameter_name`
- `parameter_type`
- `data_type`
- `enum_value`
- `value_range`
- `default_value`
- `description`
- `is_required`
- `relationship_notes`

`parameters` may be an empty array only when the knowledge base does not provide enough evidence for parameter extraction. n8n treats this as `no_parameters_found` and does not write parameter rows.

## n8n Validation

n8n validates before writing:

- The response is parseable JSON.
- The root value is an object.
- `parameters` is an array.
- Every parameter has a non-empty `parameter_name`.
- `is_required` is normalized to boolean.
- String fields are normalized to strings.
- Duplicate `parameter_name` entries are deduplicated.
- For duplicates, n8n keeps the entry with more non-empty fields.
- If the response contains fields outside the schema, n8n ignores them.
- If `parameters` is empty, n8n records `no_parameters_found`.

## Failure Handling

Failures are not written to `command_parameters`.

Retry once when:

- SSE finishes but the final `answer` is not valid JSON.
- JSON is valid but schema validation fails.
- The response lacks `parameters`.
- The stream closes before a `complete` event.

Do not retry when:

- Xelora returns an authorization error.
- The target agent or knowledge base is missing.
- The request body is invalid.

Failure records should include:

```json
{
  "command_id": 123,
  "command": "SORTD",
  "stage": "xelora_parameter_parse",
  "status": "failed",
  "error_type": "invalid_json",
  "raw_response": "...",
  "attempt_count": 2,
  "created_at": "2026-07-16T00:00:00Z"
}
```

The exact failure table can reuse an existing workflow log table if one exists. If not, create a small n8n-side failure log table during implementation planning.

## Later Skills

Parameter parsing intentionally avoids a skill in phase one.

Later phases should introduce:

- `verification-case-table-generator`: produces structured verification case rows from parameter rows, return-code data, and command context.
- `toin-fs-code-generator`: produces TOIN&FS CL, COBOL, and PF/SF file content from structured case rows and project parameters.

These skills should hold stable business rules such as case categories, priority rules, return-code handling, `PGM_BASE_CODE`, `PGM_FILE_SEQ`, `SRTEST`, file naming, error handling templates, and disallowed CL syntax.

## Success Criteria

The parallel workflow is successful when:

- n8n can parse at least one known CL command through Xelora without passing `source_content`.
- Xelora uses `Manual_ASP` and returns strict JSON.
- n8n can validate and write parameter records to an isolated target.
- Invalid JSON is retried once and then recorded as failure.
- Empty evidence does not produce fabricated parameter rows.
- Existing Dify-backed workflow remains unchanged and available during comparison.
- Xelora output can be compared against the existing workflow output before any production cutover decision.

## Out Of Scope

- Migrating verification case table generation.
- Migrating TOIN&FS code generation.
- Building new skills.
- Changing the `Manual_ASP` ingestion pipeline.
- Reworking current database schemas beyond minimal logging needs.
- Replacing n8n orchestration.
- Replacing, disabling, or deleting the existing Dify/RAGFlow workflow.
