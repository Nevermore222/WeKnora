# Xelora CL 技术点检证用例三文件代码生成设计

## 状态

已确认设计。目标是新建独立的 CL 技术点直出工作流和 Xelora 智能体，不修改旧的 CL/COBOL 工作流或 `CL检证用例代码生成器 TOIN&FS` 智能体。

## 目标与范围

- 同时支持 TOIN（`project_id=1`）和 Fukutsu（`project_id=3`）。
- 输入为 CL `tech_point` 数据，按命令和技术点抽取 SOURCE 证据。
- 每个去重后的 SOURCE 结构生成一个正常系和两到三个边界值用例。
- 每个技术点生成完整的 `.CL`、`.COBOL/.CBL`、`.SF/.PF` 三文件。
- 保留原始 SOURCE 抽取记录和代码生成来源快照，成功记录排除后续重复处理。
- Fukutsu 重新生成不能覆盖 TOIN 的同命令结果。

## 入口契约

Webhook 接收 `project_id`、`command_id`，`tech_point_no` 可选，`batch_size` 可选。未指定技术点时处理该命令下所有未完成技术点；指定时只处理该技术点。`project_id` 必须是 1 或 3，`command_id` 必须明确传入。

## 数据查询与抽样

查询字段固定包含 `command_name`、`pgmid`、`tech_point_no`、`sub_category`、`source_content`、`variable_declaration`，只读取对应项目和命令的 `language='CL'` 行，并显式排除 `MOVE`、`IF`。

技术点分组键为 `project_id + language + command_name + tech_point_no + sub_category`。组内按 `pgmid ASC NULLS LAST, tech_point.id ASC` 排序并按 PGM 分桶，轮询各桶：先覆盖不同 PGM，再按排序补齐到最多五条；候选少于五条时全部传入。每个技术点独立构造 payload、去重、生成和归档，变量定义不跨技术点复用。

## 审计表

新增三张 `analyzes` 表：

1. `cl_tech_point_case_references` 保存抽取过的原始 SOURCE 行及字段快照。唯一键为项目、语言、命令、技术点、小分类和 `source_tech_point_id`。只有 `completed` 行参与后续排除，失败行保留并可重试。
2. `cl_tech_point_code_generations` 每个项目、语言、命令、技术点一行，保存 `allocated/generating/completed/failed` 状态、程序编号、三文件路径、智能体信息、错误和时间戳，并以项目维度建立完成唯一键。
3. `cl_tech_point_code_generation_sources` 保存实际传给代码智能体的五条 SOURCE 快照、PGM、变量定义、结构指纹、结构 ID、抽取顺序和生成批次，作为 SOURCE 代码编写备份。

成功顺序为三文件保存和规范化成功后，在同一事务中标记生成完成、写入来源快照并将引用标记为完成；失败只记录失败状态，不消费引用。

## Xelora 智能体

新智能体接收一个技术点和最多五条 `references`。变量定义签名只比较层级、类型、长度/精度、初始化和数组属性，忽略变量名。SOURCE 结构签名保留命令、参数顺序、嵌套/续行结构及变量/数字/字符串/系统常量/资源名等值类别，忽略变量名和具体数值。只有两个签名同时一致才去重，去重只发生在同一技术点内。

每个保留结构生成一个 `正常系` 和两到三个有依据的 `境界値`，不得生成无依据的异常、性能或跨结构用例。非空 `variable_declaration` 必须按真实定义生成；为空时沿用旧逻辑，但不得借用其他技术点的定义。

输出严格为三个源码段，顺序为：

```text
## TOINSRC/{PROGRAM_BASE}C{SEQ}.CL
## TOINSRC/{PROGRAM_BASE}B{SEQ}.COBOL
## TOINSRC/{PROGRAM_BASE}F{SEQ}.SF
```

CL 注释必须包含完整 `tech_point_no`、所有贡献 PGM、`source_tech_point_id`、`structure_id` 和用例类型。三类源码继承旧智能体的 CL v10、日文、固定格式 COBOL、PF/SF `/END` 和 72 显示列规则；`TOINSRC` 仅为解析标题，不创建物理目录。

## 文件与项目隔离

程序前缀为 `CO`（TOIN）和 `CF`（Fukutsu），编号示例为 `CO001C01/CO001B01/CO001F01` 或 `CF001C01/CF001B01/CF001F01`。实际目录直接位于项目目录下，不创建 `TOINSRC`、`QCLSRC`、`QCBLSRC`、`QFILESRC` 层级，例如：

```text
/home/n8n/.n8n/verification_cases_code/Fukutsu/ASP/CL/{command_id}_{command_name}/{timestamp}/
```

三文件全部成功并通过规范化后才完成登记；项目唯一键和项目目录保证 Fukutsu 不覆盖 TOIN 同命令文件。

## 工作流阶段

1. 初始化审计表并校验入口参数。
2. 查询项目、命令和待处理技术点。
3. PGM 优先抽取最多五条 SOURCE。
4. 分配或恢复稳定程序编号。
5. 逐技术点创建 Xelora 会话并调用新智能体。
6. 解析 SSE，验证三段代码和技术点追踪注释。
7. 创建项目隔离目录，保存 staging 文件，调用 Shift_JIS/CRLF 规范化服务。
8. 校验三文件、行宽、格式和命名。
9. 成功事务登记生成记录、来源备份和抽取完成状态。
10. 继续批处理并返回成功/失败明细。

## 错误处理与验证

缺少项目/命令、项目不存在、技术点缺字段、智能体 SSE 不完整、源码缺段、文件名不匹配、去重后缺少正常/边界用例、跨技术点变量复用、超过 72 列或规范化失败时拒绝完成登记。失败技术点可重试，已完成技术点幂等跳过。

测试覆盖五 PGM 抽样、少于五 PGM 补齐、候选不足、已完成排除、两项目隔离、去重条件、变量定义回退、三文件解析、CL 追踪注释、行宽、固定格式 COBOL、规范化失败和事务回滚。
