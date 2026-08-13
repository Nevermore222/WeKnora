# TOIN CL 技术点引用抽样与二维表生成 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 改造 Xelora 的 CL 技术点二维表生成链路，使每个技术点按 PGM 覆盖优先抽取最多五条源记录，并在 case 成功写入后幂等登记引用，供后续排除重复处理。

**Architecture:** 新增 PostgreSQL 引用登记表作为消费日志；工作流在 SQL/Code 节点完成稳定抽样，将完整引用集合传给智能体；case upsert 与引用登记在同一事务内完成。现有 Dify 智能体 YAML 与 n8n JSON 保留输入输出契约，只扩展 references 和来源元数据。

**Tech Stack:** PostgreSQL migration SQL, n8n Code/Postgres nodes, Dify YAML, Node.js validation scripts, Node test runner.

---

### Task 1: 创建引用登记表迁移

**Files:**
- Create: `E:\Xelora\WeKnora\scripts\n8n\migrations\2026-08-13-cl-tech-point-case-references.sql`
- Create: `E:\Xelora\WeKnora\scripts\n8n\validate-toin-cl-tech-point-references.mjs`

- [ ] **Step 1: Write the failing validation test**

在验证脚本中连接 PostgreSQL，查询 `to_regclass('analyzes.cl_tech_point_case_references')`，不存在时以非零状态退出；同时断言唯一约束和外键存在。

- [ ] **Step 2: Run validation to verify it fails**

运行：`node scripts/n8n/validate-toin-cl-tech-point-references.mjs`

预期：失败并报告登记表不存在。

- [ ] **Step 3: Write migration**

创建设计文档规定的字段、外键和唯一约束；使用 `CREATE TABLE IF NOT EXISTS`，并为 `(project_id, language, command_name, tech_point_no, sub_category)` 和 `source_tech_point_id` 建查询索引。

- [ ] **Step 4: Apply migration and rerun validation**

运行：`Get-Content scripts/n8n/migrations/2026-08-13-cl-tech-point-case-references.sql | docker exec -i -e PGPASSWORD=postgres Xelora-app psql -v ON_ERROR_STOP=1 -h 192.168.8.247 -p 5432 -U postgres -d n8n`

预期：迁移成功，验证脚本报告表、唯一约束和外键均存在。

- [ ] **Step 5: Commit**

```powershell
git add scripts/n8n/migrations/2026-08-13-cl-tech-point-case-references.sql scripts/n8n/validate-toin-cl-tech-point-references.mjs
git commit -m "feat: add CL tech point case reference ledger"
```

### Task 2: 实现并测试 PGM 覆盖优先抽样

**Files:**
- Create: `E:\Xelora\WeKnora\scripts\n8n\toin-cl-reference-selection.mjs`
- Create: `E:\Xelora\WeKnora\scripts\n8n\toin-cl-reference-selection.test.mjs`

- [ ] **Step 1: Write failing tests**

覆盖四种输入：五个 PGM 各取一条；两个 PGM 轮询补足五条；候选少于五条全取；已登记 `source_tech_point_id` 被排除。断言输出顺序稳定且每条保留完整源字段。

- [ ] **Step 2: Run tests to verify failure**

运行：`node --test scripts/n8n/toin-cl-reference-selection.test.mjs`

预期：因选择函数尚不存在而失败。

- [ ] **Step 3: Implement minimal selector**

导出 `selectReferences(rows, consumedIds, limit = 5)`：按技术点分组，按 `pgmid || ''` 和 numeric `id` 排序，构建 PGM 桶并轮询每个桶；先过滤 consumed IDs；返回 `{ group, references }[]`。

- [ ] **Step 4: Run tests to verify pass**

运行同一 `node --test` 命令，预期全部通过。

- [ ] **Step 5: Commit**

```powershell
git add scripts/n8n/toin-cl-reference-selection.mjs scripts/n8n/toin-cl-reference-selection.test.mjs
git commit -m "feat: select CL tech point references by PGM coverage"
```

### Task 3: 更新 CL 智能体输入契约和提示词

**Files:**
- Create: `E:\Xelora\n8n\二维表\CL命令检证用例生成智能体(基于tech_point)_xelora.yml`
- Test: `E:\Xelora\WeKnora\scripts\n8n\validate-toin-cl-agent-config.mjs`

- [ ] **Step 1: Write failing config validation**

验证 YAML/文本包含 `references`、`source_tech_point_id`、`PGMID`、`SOURCE_CONTENT`、`variable_declaration`、类型/长度/精度保真规则，以及禁止拼接不存在 SOURCE 的规则。

- [ ] **Step 2: Run validation to verify failure**

预期：旧智能体配置缺少 references 契约，验证失败。

- [ ] **Step 3: Create revised agent config**

以现有 YAML 为基线，保留模型、数据集和保存工具；将输入改为 `TECH_POINT` + `references[]`，要求输出 JSON 包裹 Markdown 二维表和 `used_reference_ids[]`，并明确变量定义、SOURCE 结构、case 可执行性约束。

- [ ] **Step 4: Run config validation**

预期：验证脚本通过，并确认旧文件未被覆盖。

- [ ] **Step 5: Commit**

```powershell
git add "E:\Xelora\n8n\二维表\CL命令检证用例生成智能体(基于tech_point)_xelora.yml" scripts/n8n/validate-toin-cl-agent-config.mjs
git commit -m "feat: define CL case agent reference contract"
```

### Task 4: 重写 n8n 工作流

**Files:**
- Create: `E:\Xelora\n8n\二维表\CL命令检证用例二维表工作流（tech_point同款输入保存+SOURCE用例智能体）_xelora_1.4.0.json`
- Create: `E:\Xelora\WeKnora\scripts\n8n\validate-toin-cl-reference-workflow.mjs`

- [ ] **Step 1: Write failing workflow validation**

解析 JSON 并断言存在：引用表查询、PGM 轮询抽样、references 传参、case 成功分支、引用登记 SQL、事务/回滚处理和 `ON CONFLICT DO NOTHING`。

- [ ] **Step 2: Run validation to verify failure**

预期：旧工作流没有新登记表和 PGM 抽样节点，验证失败。

- [ ] **Step 3: Build revised workflow**

复制现有工作流并替换：

- tech_point 查询增加 `project_id=1 AND language='CL' AND command_name NOT IN ('MOVE','IF')`；
- 查询已登记 `source_tech_point_id` 并传给 selector；
- 按技术点输出最多五条 `references`；
- 智能体响应解析必须返回 `used_reference_ids`；
- case upsert 与引用登记使用同一批次 UUID；
- case 解析为空或写入失败时不登记引用；
- Postgres 节点使用显式 `BEGIN/COMMIT`，异常路径 `ROLLBACK`；
- 引用登记使用 `ON CONFLICT DO NOTHING`。

- [ ] **Step 4: Run static workflow validation**

运行：`node scripts/n8n/validate-toin-cl-reference-workflow.mjs "E:\Xelora\n8n\二维表\CL命令检证用例二维表工作流（tech_point同款输入保存+SOURCE用例智能体）_xelora_1.4.0.json"`

预期：节点、连接、SQL 关键字和引用字段全部通过。

- [ ] **Step 5: Commit**

```powershell
git add "E:\Xelora\n8n\二维表\CL命令检证用例二维表工作流（tech_point同款输入保存+SOURCE用例智能体）_xelora_1.4.0.json" scripts/n8n/validate-toin-cl-reference-workflow.mjs
git commit -m "feat: generate CL cases from sampled tech point references"
```

### Task 5: 集成验证与回归检查

**Files:**
- Modify: `E:\Xelora\WeKnora\scripts\n8n\validate-toin-cl-reference-workflow.mjs`
- Create: `E:\Xelora\WeKnora\scripts\n8n\toin-cl-reference-integration.test.mjs`

- [ ] **Step 1: Add integration fixtures**

使用内存 fixture 模拟两个技术点、多个 PGM、部分已消费行和智能体 `used_reference_ids`，验证输出 case 与登记行一一对应。

- [ ] **Step 2: Verify rollback behavior**

模拟 case 写入失败，断言登记集合为空；模拟重复运行，断言 case key 和登记唯一键不增加。

- [ ] **Step 3: Run complete verification**

运行：

```powershell
node --test scripts/n8n/toin-cl-reference-selection.test.mjs scripts/n8n/toin-cl-reference-integration.test.mjs
node scripts/n8n/validate-toin-cl-tech-point-references.mjs
node scripts/n8n/validate-toin-cl-agent-config.mjs
node scripts/n8n/validate-toin-cl-reference-workflow.mjs "E:\Xelora\n8n\二维表\CL命令检证用例二维表工作流（tech_point同款输入保存+SOURCE用例智能体）_xelora_1.4.0.json"
```

预期：所有测试和验证命令退出码为 0。

- [ ] **Step 4: Commit**

```powershell
git add scripts/n8n
git commit -m "test: verify TOIN CL reference sampling workflow"
```

