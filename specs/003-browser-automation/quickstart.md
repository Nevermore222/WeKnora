# Quickstart: Validate Browser Automation Provider Path

This quickstart validates the product contract for the browser automation feature.

## 1. Trigger A Browser Navigation Task

From an agent-enabled conversation:

1. Ask the agent to open a browser and navigate to a public URL.
2. Wait for the task to complete.

Expected:

- The `browser_navigate` tool is invoked with the URL and capture mode.
- The gateway dispatches a browser task through the configured provider.
- The job status transitions through running to succeeded or failed.

## 2. Verify Artifact Registration

1. Check the conversation for the response after the browser task completes.

Expected:

- A screenshot (PNG) or page content (HTML/Markdown) artifact is listed in the tool result.
- The artifact is downloadable from the chat context.
- The artifact has the correct kind (`image` for screenshots, `markdown` for HTML/content).

## 3. Verify Workspace-Bound Artifact Routing

1. Create a workspace-bound conversation.
2. Trigger a browser navigation task.

Expected:

- The screenshot or content file is written inside the bound workspace root.
- The artifact record carries the conversation's workspace ID.
- The artifact is visible in the same artifact list as file-producing skills.

## 4. Verify Unbound Conversation Fallback

1. Create a conversation without a workspace binding.
2. Trigger a browser navigation task.

Expected:

- The artifact is written to the skill-private base path (fallback behavior).
- No workspace boundary error is raised.
- The artifact is still registered and downloadable.

## 5. Verify Boundary Enforcement

1. In a workspace-bound conversation, simulate a path escape on the browser artifact output.

Expected:

- The write is blocked before file creation.
- A clear error is returned explaining the path escape.
- No artifact is created outside the workspace boundary.

## 6. Verify Provider Error Handling

1. Configure an invalid browser provider name or make the browser binary unavailable.
2. Trigger a browser navigation task.

Expected:

- The system returns a structured error explaining the provider is unavailable.
- The job status is `failed` with a clear error message.
- No silent failure or hidden fallback artifact.

## 7. Verify Timeout Handling

1. Set a short browser timeout (e.g., 5 seconds).
2. Navigate to a slow-loading page.

Expected:

- The task fails with a timeout error after the configured duration.
- The error message clearly indicates a timeout occurred.
- Partial results (if any) are still registered.
