# TOIN CL 技术点引用抽样与二维表生成设计

## 目标

基于 `analyzes.tech_point` 中 TOIN 项目（`project_id = 1`）的 CL 技术点，生成可用于未来真实 CL 检证代码编写的二维表 case。每个技术点最多选择五条源记录作为证据集合，不要求每条源记录分别生成 case。成功写入 `analyzes.cl_verification_cases_master_proj` 后，登记本次使用的源记录，后续运行排除已登记记录。

## 范围与过滤

- 仅处理 `project_id = 1`、`language = 'CL'` 的技术点。
- `MOVE` 和 `IF` 已在技术点导入阶段排除；工作流仍需在查询层显式排除，防止历史脏数据进入生成流程。
- 技术点分组键为 `project_id + language + command_name + tech_point_no + sub_category`。
- 同一技术点的不同 `SOURCE_CONTENT`、`PGMID`、变量定义均视为候选源记录。

## 抽样算法

1. 从 `analyzes.tech_point` 读取候选记录，并左连接抽取登记表，排除已经登记的 `source_tech_point_id`。
2. 每个技术点内按 `PGMID ASC NULLS LAST, tech_point.id ASC` 排序，并按 PGM 分桶。
3. 采用轮询选择：第一轮每个 PGM 取第 1 条；不足五条时，第二轮每个 PGM 取第 2 条，依次类推，直到选择五条或候选耗尽。
4. PGM 数大于等于五时，第一轮即得到五个不同 PGM；PGM 数少于五时，先覆盖所有 PGM，再用同一排序从各 PGM 补齐。
5. 候选不足五条时全部选择。选择顺序必须稳定，确保重试结果可复现。

## 抽取登记表

新增表：`analyzes.cl_tech_point_case_references`。

字段：

- `id bigserial primary key`
- `project_id integer not null`
- `language varchar(20) not null`
- `command_id bigint null`
- `command_name varchar(256) not null`
- `tech_point_no varchar(256) not null`
- `sub_category varchar(256) not null`
- `source_tech_point_id bigint not null references analyzes.tech_point(id)`
- `pgmid varchar(256) null`
- `source_content text null`
- `variable_declaration text null`
- `extraction_batch_id uuid not null`
- `created_at timestamptz not null default current_timestamp`

唯一约束：`(project_id, language, command_name, tech_point_no, sub_category, source_tech_point_id)`。

登记表保存源字段快照，便于审计当时传给智能体的内容；`source_tech_point_id` 是排除和幂等判断的权威字段。

## 工作流事务边界

每个技术点批次使用一个 PostgreSQL 事务：

1. 写入二维表 case，使用现有 case 唯一键执行 upsert。
2. 只有 case 写入成功后，才插入本批次引用记录，使用 `ON CONFLICT DO NOTHING`。
3. 任一步骤失败则回滚 case 和引用登记，重试时仍可重新抽样。
4. case 写入与引用登记必须使用同一批次 UUID，便于追踪和恢复。

## 智能体输入契约

智能体接收一个技术点及最多五条 `references`，每条包含：

`source_tech_point_id, pgmid, command_name, tech_point_no, sub_category, source_content, variable_declaration`。

生成要求：

- 保留每条 `SOURCE_CONTENT` 的原始片段结构、顺序和关键语法，不把多个片段拼成不存在的源码。
- `variable_declaration` 非空时，按原定义中的变量名、类型、长度、精度、初始化/数组属性生成参数；为空时不得臆造变量定义。
- `parameter_config` 明确变量到输入值的映射和类型；`input_data` 给出可执行的输入；`expected_result` 给出可验证的结果和状态；`notes` 记录引用的 PGM 与源行 ID。
- 每个 case 必须带 `tech_point_no`，并在 notes 或元数据中保留引用来源，保证未来可反向生成真实检证代码。
- 五条引用是证据集合；智能体按需要生成覆盖正常、边界、异常和兼容性的 case，但不得为了数量复制无差异 case。

## 错误处理与幂等

- 技术点缺少 `tech_point_no`、`sub_category` 或 `command_name` 时记录错误并跳过该技术点，不登记引用。
- 智能体输出无法解析为完整二维表时不写入 case，也不登记引用。
- 已登记源行不再进入候选；技术点仍有未登记源行时继续处理剩余候选。
- 对同一技术点并发执行时，按技术点分组加事务锁，避免两个批次同时消费相同源行。

## 验收标准

- 抽样测试覆盖：五个 PGM 各取一条；少于五个 PGM 时轮询补齐；候选不足五条时全取；已登记行排除；排序稳定。
- 数据库测试确认唯一约束、外键、事务回滚和重复执行幂等。
- 工作流验证确认 case 写入失败时引用表没有新增记录，case 成功后引用表数量等于本批次实际使用源行数。
- 智能体样例验证确认非空变量定义的类型/长度保留，空变量定义不被生成，SOURCE 片段结构未被破坏。

