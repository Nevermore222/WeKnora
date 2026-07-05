# xelora — Xelora CLI

A command-line interface for the Xelora RAG knowledge-base server. Lets you
authenticate, manage knowledge bases and documents, run hybrid search, and
ask streaming RAG questions from your terminal or from an AI agent.

```bash
$ xelora --help
Command-line client for the Xelora RAG server. Manage knowledge bases
and documents, run hybrid search, chat with grounded answers, or expose
a curated read-only MCP tool surface for AI agents.

Available Commands:
  agent       Manage custom agents (CRUD + status/check)
  api         Make a raw API request to the Xelora server
  auth        Manage authentication credentials and profiles
  chat        Ask a streaming RAG question against a knowledge base
  chunk       Manage document chunks (RAG retrieval debug)
  completion  Generate the autocompletion script for the specified shell
  profile     Manage CLI profiles (named connection targets)
  doc         Manage documents in a knowledge base
  doctor      Run 4 self-checks: base URL, auth, server version, credential storage
  help        Help about any command
  kb          Manage knowledge bases
  link        Bind the current directory to a knowledge base
  mcp         Run xelora as a Model Context Protocol server
  search      Search across chunks, knowledge bases, documents, or sessions
  session     Manage chat sessions
  unlink      Remove the directory's knowledge-base binding
  version     Show CLI build metadata
```

The wire contract for AI agents is documented [below](#wire-contract).
For contributing to the CLI source, see [AGENTS.md](AGENTS.md).

---

## Install

### From source

Requires Go 1.26+.

```bash
git clone https://github.com/Tencent/Xelora.git
cd Xelora/cli
go build -o xelora .
sudo mv xelora /usr/local/bin/   # or anywhere on $PATH
```

### Pre-built binaries

Pre-built binaries for Linux / macOS / Windows are produced by CI on each
release. Grab the latest from the [Releases page](https://github.com/Tencent/Xelora/releases).

---

## 5-minute quickstart

```bash
# 1. Register your Xelora server as a profile and make it active
xelora profile add prod --host https://kb.example.com --use

# 2. Authenticate the active profile (interactive password prompt)
xelora auth login

# 2b. Or pipe an API key from stdin (for CI / AI agents)
echo "sk-..." | xelora auth login --with-token

# 3. List knowledge bases
xelora kb list

# 4. Bind this directory to a knowledge base — subsequent commands auto-resolve --kb
xelora link --kb my-knowledge-base

# 5. Upload a document, then block until parsing finishes
xelora doc upload notes.md
xelora doc wait doc_abc                          # exit 0 completed, 1 failed, 124 --timeout, 130 ^C

# 6. Search
xelora search chunks "what is reciprocal rank fusion?"

# 7. Ask the LLM (streams to terminal)
xelora chat "summarise the design doc"

# 8. Manage custom agents and run them (see `xelora agent --help` / `xelora session --help`)
xelora agent list
xelora session ask --agent ag_abc "what's our q4 retention plan?"

# 9. Inspect a document's chunks for RAG retrieval debug
xelora chunk list --doc doc_xyz

# 10. Health & verification verbs
xelora kb status kb_abc       # fast snapshot: reachable / counts / processing flag (1 HTTP)
xelora kb check kb_abc        # deep verify: also aggregates failed_count via doc list (1+N HTTP)
xelora agent status ag_abc    # fast: reachable / model_id
xelora agent check ag_abc     # deep: probes every KB in the agent's scope
```

---

### Agent quick start

For AI agents (Claude Code, Cursor, Gemini CLI, etc.) integrating Xelora:

1. Install: `brew install xelora` or `go install github.com/Tencent/Xelora/cli@latest`
2. Register a profile, then authenticate it (background; extract login URL for the user):
   ```bash
   xelora profile add prod --host <server-url> --use
   xelora auth login
   ```
3. Register MCP in the host's MCP config:
   ```json
   {"mcpServers": {"xelora": {"command": "xelora", "args": ["mcp", "serve"]}}}
   ```
4. Read the [wire contract](AGENTS.md#wire-contract-for-ai-agents) before
   parsing `--format json` output.
5. Read the [exit-10 anti-patterns](AGENTS.md#exit-10-anti-patterns) before
   any destructive call.

**Bundled Agent Skills.** This CLI ships [Agent Skills](https://agentskills.io/specification)
under [`skills/`](skills/) that teach an agent to drive Xelora without trial and error:

- [`xelora-shared`](skills/xelora-shared/SKILL.md) — **read first**: auth/profile
  sequence, `--kb` resolution, the JSON-envelope + exit-code contract, the exit-10
  protocol, `--dry-run`, and CLI-vs-MCP selection.
- [`xelora-rag-search`](skills/xelora-rag-search/SKILL.md) — when to use `chat`
  vs `session ask` vs `search chunks`, plus retrieval gotchas.

MVP install: symlink them into your agent's skills directory (from a source checkout):

```bash
ln -s "$PWD/skills/xelora-shared"     ~/.claude/skills/xelora-shared
ln -s "$PWD/skills/xelora-rag-search" ~/.claude/skills/xelora-rag-search
```

Each skill's frontmatter records the CLI version it was `tested_against`; a CI
parity test (`internal/skillparity`) fails if a skill ever references a command,
flag, or MCP tool the CLI no longer has. (A `xelora skills install` command is
planned; for now, symlink or copy.)

---

## Multi-profile

`profile.*` manages profile *records* (positional `<name>`); `auth.*` operates
on the *active* profile (override per-invocation with the global `--profile`
flag). Create a profile first, then authenticate it:

```bash
xelora profile add prod    --host https://prod.example.com --use     # add + switch
xelora auth login                                                    # authenticate active (prod)

xelora profile add staging --host https://staging.example.com        # add (stays inactive)
echo "sk-..." | xelora --profile staging auth login --with-token     # authenticate staging

xelora auth list
xelora profile use prod                                              # switch back
```

Credentials are persisted to your OS keyring (Keychain on macOS, libsecret on
Linux, Wincred on Windows) when available, otherwise to a 0600-mode file
under `$XDG_CONFIG_HOME/xelora/secrets/`. The active profile lives in
`~/.config/xelora/config.yaml`.

To remove a profile's stored credentials:

```bash
xelora auth logout                       # active profile
xelora --profile staging auth logout     # specific profile
xelora auth logout --all
```

---

## Wire contract

Designed to be AI-agent-first. Stable across minor releases; breaking
changes announced in the changelog and the corresponding
`xelora --version` bump.

### Streams

- **stdout** is the data channel: bare JSON with `--format json`, or
  human-formatted output. Never carries error text.
- **stderr** is logs, progress, warnings, and errors. A non-empty
  stderr does **not** mean failure — read the exit code.

### JSON output

Every command supports `--format json`, emitting bare JSON for the
resource it produces — an array for `list` / `search`, a single object
for `view` and write outcomes:

```bash
xelora kb list --format json                              # [{ "id": "kb_x", "name": "Eng" }, …]
xelora kb view kb_x --format json                         # { "id": "kb_x", "name": "Eng", … }
xelora kb list --format json --jq '.[] | {id, name}'      # project to listed fields
xelora kb list --format json --jq '.[].id'                # jq over the bare data
```

`--format ndjson` is also accepted for streaming list commands; each
element is emitted as its own JSON line. `--format json` is the default
regardless of TTY — running `xelora kb list | jq` works without an
explicit flag. Use `--format text` for human-readable output.

### Errors

On failure, stdout stays empty and the typed error goes to stderr in
this format:

```
<code.namespace>: <message>[: <wrapped cause>]
hint: <actionable next-step>
```

Example:

```
auth.unauthenticated: fetch current user: HTTP error 401: ...
hint: run `xelora auth login`
```

The full code registry is in `cli/internal/cmdutil/errors.go`
(`AllCodes()`). Code namespaces: `auth.*` / `resource.*` / `input.*` /
`server.*` / `network.*` / `local.*` / `mcp.*` / `operation.*` (CLI-level
wait/poll outcomes: `operation.timeout`, `operation.failed`, `operation.cancelled`).

### Exit codes

| Code | Meaning | Agent action |
|---|---|---|
| `0`   | success                                                | continue |
| `1`   | typed `local.*` / `operation.failed` / unclassified    | read stderr, decide retry/abort |
| `2`   | flag / argument validation error                       | re-check `xelora <cmd> --help` |
| `3`   | `auth.*` (token missing / expired / forbidden)         | re-auth, then retry |
| `4`   | `resource.not_found`                                   | verify the resource id |
| `5`   | `input.*` (other than `confirmation_required`)         | adjust args, retry |
| `6`   | `server.rate_limited`                                  | back off, retry |
| `7`   | `server.*` / `network.*`                               | transient — retry with backoff |
| `10`  | **`input.confirmation_required`** (high-risk write)    | ask the human, retry with `-y` only after explicit approval |
| `124` | `operation.timeout` (e.g. `doc wait --timeout` reached) | raise `--timeout` or check the underlying job |
| `130` | `operation.cancelled` (SIGINT / SIGTERM)               | stop, do not retry |

**Exit 10** is the wire-level signal for "destructive write needs
explicit confirmation". Pass `-y/--yes` on `kb delete` /
`doc delete` (including `--all --kb=<id>`) / `session delete` /
`profile remove` (on the current profile) / `agent delete` /
`chunk delete` when running headless.
**Never auto-add `-y` without the user's explicit go-ahead** — exit 10
is the guard against unintended writes.

### Other AI-agent ergonomics

- For chat / session ask in AI-agent contexts, pass `--format json` —
  streaming tokens to stdout makes JSON parsing impossible.
- `--format json` composes with the global `--profile <name>` for
  single-shot profile overrides without disk writes.
- `xelora mcp serve` exposes a curated read-only tool surface over
  stdio MCP for any MCP-compatible client.

---

## Advanced operations not exposed as flags

Xelora CLI exposes top use cases as polished commands; deep
configuration goes through the raw HTTP passthrough. CLI flag coverage
targets common workflows, not 1:1 API parity. Examples of deep
operations that intentionally go through `xelora api`:

- **Tuning a KB's nested config** — chunking strategy, summary model,
  multimodal extraction defaults, FAQ thresholds, VLM model. Use
  `xelora api PUT /api/v1/knowledge-bases/<id> --input -` with a JSON
  body matching the server's `UpdateKnowledgeBaseRequest`. (Note: the
  storage provider is set once at create time via
  `kb create --storage-provider <name>` and is not updatable.)
- **Per-request `chat` parameters** — multi-KB scope, summary model
  override, image attachments, web search toggle. Use `xelora api POST
  /api/v1/knowledge-chat/<session-id> --input -`.
- **Per-request `session ask --agent` overrides** — same shape via
  `xelora api POST /api/v1/agent-chat/<session-id> --input -`.
- **Operations without a CLI verb** — register / change-password /
  OIDC flows, organization / sharing endpoints, tenant management.

`xelora api --help` documents the raw passthrough. Run
`xelora doctor` first to verify auth and base URL.

---

## Dry-run preview

Add `--dry-run` to any mutation command to preview the would-be action without executing it. Useful for verifying flag/arg parsing before committing to a destructive operation, or for agent-side action planning.

```bash
# Preview a kb create without actually creating
xelora kb create --name "test-kb" --description "for review" --dry-run

# Output (single line; pretty-printed here for readability):
# {
#   "ok": true,
#   "meta": {
#     "dry_run": true,
#     "plan": {
#       "action": "kb.create",
#       "args": {"name": "test-kb", "description": "for review"}
#     }
#   }
# }
# Exit code: 0
```

dry-run is **offline**: no network calls, no file IO, no credential touches. Works without an active profile.

For destructive commands, dry-run does NOT trigger the exit-10 confirmation flow:

```bash
xelora kb delete kb_xxxx --dry-run   # exit 0, no prompt
xelora kb delete kb_xxxx             # exit 10, prompts for -y
```

For the `api` command, dry-run requires explicit write method (POST/PUT/PATCH/DELETE); GET returns FlagError:

```bash
echo '{"name":"foo"}' | xelora api -X POST /api/v1/knowledge-bases --input - --dry-run   # OK
xelora api /api/v1/knowledge-bases --dry-run                                              # exit 2: requires explicit -X
```

---

## Resuming streams

The `xelora session continue-stream` command resumes an SSE event stream for an existing assistant message. Useful for network-blip recovery or polling long-running agent invocations:

```bash
# Original streaming call captures session_id + message_id from init event:
xelora session ask "..." --agent ag_xxxx --format ndjson | tee /tmp/stream.ndjson
# {"type":"init","session_id":"sess_abc","message_id":"msg_xyz"}
# ... events flow ...
# [network blip]

# Resume the same stream:
xelora session continue-stream sess_abc --message msg_xyz
# Server REPLAYS all stored events from the start, then tails new ones.
# Agent must dedupe (by message_id or event hash) to avoid double-processing.
```

Server-side buffer TTL: 1 hour for redis mode; process lifetime for memory mode (default). After TTL, expect `local.sse_stream_aborted` typed error.

See `cli/AGENTS.md` "Stream recovery" section for the full agent contract.

---

## Health check

Run `xelora doctor` for a 4-status diagnostic (OK / warn / fail /
skip) covering base URL reachability, authentication, server-CLI
version skew, and credential storage backend. Add `--format json` for
machine-readable output, `--offline` to skip network checks.

For per-resource verification, the `status` / `check` verb pair gives
a fast vs deep choice:

| Verb | Cost | Use |
|---|---|---|
| `xelora kb status <kb-id>`     | 1 HTTP    | live counts / processing flag |
| `xelora kb check <kb-id>`      | 1+N HTTP  | adds `failed_count` via doc-list page-walk |
| `xelora agent status <agent-id>` | 1 HTTP  | reachable / model_id |
| `xelora agent check <agent-id>`  | 1+N HTTP | also probes every KB in the agent's scope |

`xelora doc wait <doc-id> [<doc-id>...]` blocks until each document
reaches a terminal `parse_status` (completed or failed). Exit codes:
0 (all completed), 1 (any failed), 124 (`--timeout` reached), 130
(Ctrl-C / SIGTERM). Multi-target is polled concurrently (max 5 in
flight; pipe through `xargs -P` for more).

---

## Development

```bash
# Run unit + contract tests
go test ./...

# Run the real-server e2e suite (requires XELORA_E2E_HOST + token env vars)
go test -tags acceptance_e2e ./acceptance/e2e/...

# Static analysis
go vet ./...
```

CI (`.github/workflows/cli.yml`) runs build + unit + contract tests on Linux /
macOS / Windows × Go 1.26, path-filtered to changes under `cli/`.

---

## Contributing / Reporting issues

- **Bugs and feature requests**: file an issue at
  [github.com/Tencent/Xelora/issues](https://github.com/Tencent/Xelora/issues).
- **Security disclosures**: see the repository-level
  [SECURITY.md](../SECURITY.md). Do not file public issues for
  security findings.
- **Pull requests**: the developer guide for editing the CLI lives in
  [AGENTS.md](AGENTS.md) (build / test / command-surface design SOP /
  CRUD flag conventions). Run `go test ./... -race -count=1` and `go vet ./...`
  before submitting.

---

## License

MIT — see the repository [LICENSE](../LICENSE).
