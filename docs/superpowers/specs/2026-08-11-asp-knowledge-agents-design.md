# ASP Knowledge-Base Agents Design

**Date:** 2026-08-11

## Decision

Create two custom agents for administrator `xuwh@softroad.com.cn`
(`user_id=8b4b0fc8-b801-49eb-8d20-e3021ea0b24f`) in tenant `10000` on
`192.168.9.177`, one for each requested knowledge base:

- `ASP-JP1 专家` binds only `ASP-JP1`
  (`1cf679e6-be1f-4a63-a825-b8d4b284879a`).
- `ASP-BSP-RM 账票专家` binds only `ASP-BSP-RM账票KnowDB`
  (`f106b6d6-a268-4fc1-8fb5-15f42f408951`).

Both agents reuse the existing `SR_ASP` Progressive Agentic RAG baseline and
the active tenant models. They use selected knowledge-base retrieval,
mandatory deep reading of retrieved documents, fresh retrieval for every
domain question, and no web-search fallback. Answers follow the language of
the user's question. The agents must cite source documents and must clearly
state when the bound knowledge base does not contain enough evidence.

## Scope And Evidence

The JP1 base currently contains 15 completed documents and 36,212 chunks,
covering JP1/AJS3 and JP1/Base design, construction, operation, commands,
messages, integration, and troubleshooting. The BSP-RM base contains 45
completed documents and 8,749 chunks, covering installation, administration,
operations, message codes, report output, DURL/FormHelper, and printer or
electronic-report interfaces.

## Configuration Invariants

- `kb_selection_mode` is `selected` and `knowledge_bases` contains exactly
  one target ID.
- `agent_mode` is `smart-reasoning` and `agent_type` is `rag-qa`.
- The configuration is cloned from `SR_ASP`
  (`99a63205-a8f4-412f-a284-ebc043838e62`) before applying the documented
  agent-specific overrides.
- The active model IDs are preserved: KnowledgeQA
  `cd3394b4-99d6-4f10-9c5b-12f773add8f7`, Rerank
  `f9d119b8-1e46-478e-8379-147839170bd3`, and VLLM
  `3efff119-1faa-463f-9d29-15a8215c18dc`.
- `web_search_enabled` and `web_fetch_enabled` are false.
- Retrieval tools remain limited to the existing knowledge and reasoning
  tools; no workspace or mutation tools are added.
- Agent instructions require fresh retrieval and deep reading for every
  in-domain question, an answer in the user's language, inline citations to
  knowledge-base sources, and an explicit evidence-gap response instead of
  unsupported claims.
- Existing agents and knowledge bases are not modified. Re-running the task
  matches targets by exact name within the administrator and tenant scope,
  updates only those matches in place, and creates a target only when its
  exact scoped name does not exist.

## Provisioning Flow And Failure Handling

Provision through the existing agent API from the remote host. Obtain any
tenant credential inside the remote process and never print or persist it in
the repository or command output. Read the current `SR_ASP` configuration,
derive each target payload independently, then create or update one target at
a time. A failure on one target must not broaden the scope of the other
operation or trigger changes to the baseline agent or either knowledge base.

After every write, read the persisted target back before continuing. Treat a
duplicate exact scoped name, a missing baseline, a missing knowledge base, an
inactive required model, or a persisted invariant mismatch as a hard failure
and report it without attempting speculative repairs.

## Verification

After provisioning, verify each agent's persisted configuration, ownership,
exact single-KB scope, disabled web capabilities, active model IDs, and
visibility in the tenant agent list. Run one read-only representative domain
question per agent and confirm that the response retrieves only its bound
knowledge base, performs document reading, answers in the question language,
and includes inline source citations. Finally, use an unsupported question to
confirm that the agent states the evidence gap rather than using web content
or fabricating an answer.
