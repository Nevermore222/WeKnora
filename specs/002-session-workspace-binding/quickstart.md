# Quickstart: Validate Session Workspace Binding

This quickstart validates the product contract for the session workspace binding feature.

**Implementation notes**:

- Workspace ID format: `tenant:<tenantID>` (e.g. `tenant:1`)
- Workspace root path: {LOCAL_STORAGE_BASE_DIR}/session-workspaces/tenant-<id>/ (defaults to /data/files/session-workspaces/tenant-<id>/)
- Binding is stored as JSONB in the sessions.workspace_binding column
- The executor gateway resolves ConversationOutputContext from the binding and sets prepared.WorkDir to the workspace root so file outputs land there
- The script path stays absolute (from `File.Path`), so the script itself is always found regardless of the working directory
- Sandbox containers access the workspace via --volumes-from (the data-files named volume is shared)
- Legacy sessions (null binding) are treated as unbound and fall back to the skill-private base path

## 1. Create A Workspace-Bound Session

From the web new-chat flow:

1. Open the create-chat screen.
2. Select a workspace from the workspace picker.
3. Create the conversation.

Expected:

- The session create request carries `workspace_binding`.
- The session response returns `workspace_binding.status=bound`.
- The chat view shows that the conversation is bound to the selected workspace.

## 2. Reopen The Session

1. Leave the conversation.
2. Reopen it from the session list.

Expected:

- The same `workspace_binding` is returned by session read APIs.
- The chat view still shows the bound workspace without relying on browser local state.

## 3. Trigger A File-Producing Turn

Use a file-producing workflow such as:

- Generate a Markdown summary
- Create a spreadsheet
- Create a report artifact

Expected:

- The runtime resolves the conversation output root from the bound workspace.
- The resulting artifact record points to the bound workspace.
- The generated file path stays inside the workspace root.

## 4. Validate Failure Handling

Simulate one of these conditions:

- Bound workspace deleted
- User access removed
- Requested path escapes workspace root

Expected:

- The write is blocked before file creation.
- The user sees a clear message explaining why the binding cannot be used.
- No hidden fallback output is created in a skill-private directory for that bound conversation turn.

## 5. Validate Legacy Session Compatibility

1. Open a session created before this feature.
2. Inspect its chat context.

Expected:

- The session is treated as `unbound`.
- Normal chat remains available.
- File-producing actions require an explicit recovery path instead of pretending a workspace is already bound.


## 6. End-to-End Docker Validation

After building and deploying with `docker compose up -d --build app`:

1. **Create a workspace-bound chat**: Open the web UI, start a new chat. The frontend auto-binds to the current tenant workspace.

2. **Verify binding persistence**: Close and reopen the chat. The workspace binding bar should still show the bound workspace name.

3. **Trigger a file-producing skill**: In an agent-enabled chat, ask the agent to create a markdown file or report. Verify:
   - The file lands under `/data/files/session-workspaces/tenant-<id>/` inside the container
   - The artifact is detected and registered with the correct workspace ID

4. **Test invalid binding recovery**: Simulate by changing the workspace_id to a mismatched value via API. The chat view should show the recovery banner with a clear message.

5. **Verify legacy session compatibility**: Open a pre-existing session. It should show "No workspace bound" and chat should work normally.

## 7. Current Verified Commands

The following checks were used for the 2026-07-12 local validation pass:

```powershell
python -m unittest discover -s skills\preloaded\officecli-document-editing\scripts -p '*_test.py'
python -m unittest discover -s skills\preloaded\xlsx\scripts -p '*_test.py'
```

The browser E2E path was also checked with a workspace-bound chat that generated
an Excel workbook through the high-level `write_xlsx` action. The resulting file
was read back with `openpyxl` and verified for sheet name, headers, and row
contents.
