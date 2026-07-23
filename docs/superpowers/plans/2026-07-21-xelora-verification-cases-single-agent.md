# Xelora Verification Cases Single-Agent Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the Dify-based CL verification-case flow with a single Xelora agent that directly generates complete verification-case CASEs and removes the default evaluation stage.

**Architecture:** Reuse the existing CL verification workflow shape, but swap the external LLM boundary to Xelora session + `agent-chat`. The new generation agent will be responsible for full CASE coverage, including parameter coverage, abnormal cases, boundary cases, and return-code-driven cases, so the workflow can publish directly after generation and persistence. The evaluation agent remains available only as a reference artifact, not part of the default path.

**Tech Stack:** n8n workflow JSON, Xelora agent YAML, Node.js upsert/build scripts, PostgreSQL-backed agent config.

---

### Task 1: Define the Xelora verification-case agent contract

**Files:**
- Modify: `E:\Xelora\WeKnora\scripts\n8n\upsert-xelora-verification-case-agent.mjs`
- Create: `E:\Xelora\WeKnora\scripts\n8n\Xelora-CL-Verification-Case-Agent.env`
- Create: `E:\Xelora\n8n\二维表\Xelora版CL命令检证用例生成智能体.yml`

- [ ] **Step 1: Clone the current CL verification prompt into a Xelora-specific system prompt**

Keep the same intent as the existing generator: input is `PARAMETER_TABLE` plus return-code data, output is a complete verification-case Markdown table. Remove any wording that implies a second evaluation pass is required.

- [ ] **Step 2: Keep the default output format as a 7-column verification table**

Require `用例编号 / テスト分類 / テスト目的 / パラメータ設定 / 期待結果 / 優先度 / 備考`, with no `入力データ` column.

- [ ] **Step 3: Make the agent comprehensive enough to stand alone**

State explicitly that the agent must cover normal, abnormal, boundary, return-code-triggered, and conditional-path cases in one pass, so the workflow does not need an evaluation step to fill gaps.

- [ ] **Step 4: Upsert the agent and emit the env file**

Run:
```powershell
node .\scripts\n8n\upsert-xelora-verification-case-agent.mjs
```

Expected: a stable `XELORA_VERIFICATION_CASE_AGENT_ID` and `XELORA_TEST_MATRIX_KB_ID` in the env file.

---

### Task 2: Rewrite the workflow to call Xelora directly

**Files:**
- Modify: `E:\Xelora\n8n\二维表\CL命令检证用例二维表工作流（评估）_1.5.0_1.json`
- Modify: `E:\Xelora\WeKnora\scripts\n8n\build-xelora-verification-case-workflow.mjs`
- Modify: `E:\Xelora\WeKnora\scripts\n8n\validate-xelora-verification-case-workflow.mjs`

- [ ] **Step 1: Replace Dify nodes with Xelora session + `agent-chat` nodes**

Keep the existing parameter-table assembly and return-code inputs, but swap the external call to the new Xelora agent.

- [ ] **Step 2: Remove the evaluation branch from the default execution path**

Delete the nodes that only inspect case completeness or add secondary corrections. The generator must publish directly after the first successful Xelora response and table parse.

- [ ] **Step 3: Preserve the existing persistence contract**

Keep the final save path into the same database table and versioning flow so downstream consumers do not change.

- [ ] **Step 4: Update validation rules**

Assert that the exported workflow:
```text
contains Xelora session creation
contains Xelora agent-chat call
does not contain the default evaluation branch
still writes the verification cases into the target table
still accepts PARAMETER_TABLE / RETURN_CODE_MESSAGES_RAW / RETURN_CODE_SOURCE
```

---

### Task 3: Verify direct-generation quality

**Files:**
- Modify: `E:\Xelora\WeKnora\scripts\n8n\build-xelora-verification-case-workflow.mjs` if needed
- Test: generated workflow JSON under `E:\Xelora\n8n\二维表\`

- [ ] **Step 1: Run syntax checks**

Run:
```powershell
node --check .\scripts\n8n\upsert-xelora-verification-case-agent.mjs
node --check .\scripts\n8n\build-xelora-verification-case-workflow.mjs
node --check .\scripts\n8n\validate-xelora-verification-case-workflow.mjs
```

- [ ] **Step 2: Run workflow validation**

Run:
```powershell
node .\scripts\n8n\validate-xelora-verification-case-workflow.mjs
```

Expected: PASS, with no evaluation branch in the active path.

- [ ] **Step 3: Smoke-check the agent prompt**

Confirm the agent prompt still demands complete CASE coverage and direct output, not review comments or follow-up evaluation.

- [ ] **Step 4: Save the result**

Keep the generated workflow JSON and agent YAML in the repo tree so they can be imported into n8n without extra translation steps.

