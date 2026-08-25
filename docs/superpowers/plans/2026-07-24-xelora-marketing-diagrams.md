# Xelora 产品宣传架构图 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 生成两张面向企业管理层的中文产品宣传图，分别解释 Xelora 的企业级 AI 基盘能力和跨行业应用生态。

**Architecture:** 使用 Archify Architecture JSON 作为唯一可维护源文件，通过确定性验证和交付命令生成自包含 HTML。图一使用纵向六层能力基盘和业务域说明卡；图二使用中心辐射的四大产业集群和行业场景说明卡。两张图均使用 Classic 视觉预设、Showcase 质量门禁和无动画静态首屏。

**Tech Stack:** Archify 2.12、JSON Schema、Node.js 22、Microsoft Edge headless、HTML/SVG、PNG。

---

## 文件结构

创建以下文件：

```text
docs/marketing/
├── xelora-ai-platform-panorama.architecture.json
├── xelora-ai-platform-panorama.html
├── xelora-ai-platform-panorama.png
├── xelora-industry-ecosystem.architecture.json
├── xelora-industry-ecosystem.html
└── xelora-industry-ecosystem.png
```

职责：

- `*.architecture.json`：宣传图的事实、文案、布局、分组和关系源文件。
- `*.html`：Archify 验证后交付的自包含交互式成品。
- `*.png`：用于 PPT、方案书和宣传材料的 2560×1600 静态预览。

不修改业务源码，不提交 Git；只有用户明确要求提交时才执行版本控制操作。

---

### Task 1: 建立交付目录并验证 Archify

**Files:**
- Create directory: `docs/marketing/`

- [ ] **Step 1: 确认交付目录父路径**

Run:

```powershell
Get-Item .\docs | Select-Object FullName
```

Expected: 输出 `E:\Xelora\WeKnora\docs`。

- [ ] **Step 2: 创建宣传图目录**

Run:

```powershell
New-Item -ItemType Directory -Force .\docs\marketing
```

Expected: `docs/marketing` 存在。

- [ ] **Step 3: 验证 Archify 安装**

Run:

```powershell
node "$env:USERPROFILE\.agents\skills\archify\bin\archify.mjs" doctor
```

Expected: 结尾输出 `Archify is ready.`。

---

### Task 2: 创建“企业级知识与智能体 AI 基盘全景图”

**Files:**
- Create: `docs/marketing/xelora-ai-platform-panorama.architecture.json`

- [ ] **Step 1: 写入 Architecture JSON**

JSON 必须使用以下顶层设置：

```json
{
  "schema_version": 1,
  "diagram_type": "architecture",
  "meta": {
    "title": "Xelora 企业级知识与智能体 AI 基盘",
    "subtitle": "让企业知识可理解、业务可协同、流程可执行、决策有依据",
    "output": "docs/marketing/xelora-ai-platform-panorama.html",
    "animation": "none",
    "visual_preset": "classic",
    "quality_profile": "showcase",
    "views": [
      {
        "id": "business-value",
        "label": "从企业价值到智能应用",
        "focus": ["enterprise_value", "application_matrix"],
        "note": "从降本增效、决策提质和风险合规进入六大业务域。"
      },
      {
        "id": "intelligence-stack",
        "label": "智能体与 AI 能力",
        "focus": ["application_matrix", "agent_orchestration", "ai_capabilities"],
        "note": "查看应用如何复用 RAG、Agent、模型、检索和工具能力。"
      },
      {
        "id": "enterprise-foundation",
        "label": "Runtime、数据与治理",
        "focus": ["runtime_platform", "data_foundation", "enterprise_governance", "omnichannel_access"],
        "note": "查看私有化运行、数据底座、安全治理和多入口交付。"
      }
    ]
  }
}
```

使用以下 8 个组件，组件标签不得改成纯技术缩写：

```json
[
  {
    "id": "enterprise_value",
    "type": "cloud",
    "label": "企业价值跃迁",
    "sublabel": "知识激活 · 效率提升 · 决策增强",
    "tag": "提质 / 增效 / 控险",
    "pos": [590, 80],
    "size": [260, 78]
  },
  {
    "id": "application_matrix",
    "type": "frontend",
    "label": "企业智能应用矩阵",
    "sublabel": "六大业务域 · 36 类场景模板",
    "tag": "快速构建",
    "pos": [560, 215],
    "size": [320, 92]
  },
  {
    "id": "agent_orchestration",
    "type": "backend",
    "label": "智能体编排平台",
    "sublabel": "规划 / RAG / 工具 / MCP / 审批",
    "tag": "单智能体与多智能体",
    "pos": [585, 370],
    "size": [270, 82]
  },
  {
    "id": "ai_capabilities",
    "type": "backend",
    "label": "AI 能力中心",
    "sublabel": "模型 / 多模态 / 检索 / 图谱 / 分析",
    "tag": "统一接入",
    "pos": [585, 515],
    "size": [270, 82]
  },
  {
    "id": "runtime_platform",
    "type": "messagebus",
    "label": "Agent Runtime",
    "sublabel": "Executor / Workspace / Artifact",
    "tag": "可替换 Provider",
    "pos": [585, 660],
    "size": [270, 82]
  },
  {
    "id": "data_foundation",
    "type": "database",
    "label": "企业数据底座",
    "sublabel": "关系 / 向量 / 对象 / 实时数据",
    "tag": "企业 AI 燃料",
    "pos": [585, 805],
    "size": [270, 82]
  },
  {
    "id": "enterprise_governance",
    "type": "security",
    "label": "企业级安全治理",
    "sublabel": "租户 / RBAC / 加密 / 审计 / 观测",
    "tag": "安全可信",
    "pos": [950, 515],
    "size": [270, 82]
  },
  {
    "id": "omnichannel_access",
    "type": "external",
    "label": "全场景交付入口",
    "sublabel": "Web / Personal / API / CLI / MCP",
    "tag": "私有化与离线部署",
    "pos": [220, 515],
    "size": [270, 82]
  }
]
```

主路径只使用以下 5 条向上支撑关系：

```json
[
  {"id": "data-runtime", "from": "data_foundation", "to": "runtime_platform", "variant": "emphasis", "route": "straight"},
  {"id": "runtime-ai", "from": "runtime_platform", "to": "ai_capabilities", "variant": "emphasis", "route": "straight"},
  {"id": "ai-agent", "from": "ai_capabilities", "to": "agent_orchestration", "variant": "emphasis", "route": "straight"},
  {"id": "agent-apps", "from": "agent_orchestration", "to": "application_matrix", "variant": "emphasis", "route": "straight"},
  {"id": "apps-value", "from": "application_matrix", "to": "enterprise_value", "variant": "emphasis", "route": "straight"}
]
```

侧向关系只保留两条：

```json
[
  {
    "id": "access-apps",
    "from": "omnichannel_access",
    "to": "application_matrix",
    "label": "统一体验",
    "variant": "dashed",
    "route": "orthogonal-v"
  },
  {
    "id": "governance-runtime",
    "from": "enterprise_governance",
    "to": "runtime_platform",
    "label": "全程治理",
    "variant": "security",
    "route": "orthogonal-v"
  }
]
```

添加 9 张说明卡：

1. `企业价值`：知识资产激活、运营效率提升、决策质量增强、创新周期缩短、客户体验优化、风险合规强化。
2. `知识与智慧办公`：企业智能搜索、制度问答、文档撰写、报告生成、会议纪要、摘要翻译。
3. `营销销售与客户服务`：客户洞察、营销内容、销售陪练、商机辅助、方案生成、智能客服。
4. `运营采购与供应链`：经营分析、流程助手、采购评估、供应商审查、需求预测、库存物流问答。
5. `财务法务与风险合规`：财务分析、费用审核辅助、合同审阅辅助、政策查询、合规检查、风险预警。
6. `研发生产与质量`：研发知识、技术文档、代码辅助、故障诊断、质量分析、测试用例。
7. `人力与培训`：简历初筛、入职助手、员工政策问答、培训陪练、人才画像、绩效辅导。
8. `核心技术实现`：混合检索、ReAct、MCP、Office 产物、SSE、PostgreSQL/SQLite、Redis、DocReader。
9. `能力状态`：核心能力、可选扩展、演进方向；明确 Browser Automation 和 OpenSandbox 的状态。

卡片数组使用 Archify `cards` 结构，每个 `items` 最多 6 项，不在节点之间添加额外关系。

- [ ] **Step 2: 验证首次候选**

Run:

```powershell
node "$env:USERPROFILE\.agents\skills\archify\bin\archify.mjs" validate architecture `
  .\docs\marketing\xelora-ai-platform-panorama.architecture.json `
  --quality showcase --json
```

Expected: JSON 中 `"ok": true`、`"errors": 0`、`"warnings": 0`。

- [ ] **Step 3: 只修复验证器明确指出的问题**

若失败，读取 `diagnostics[].code`、`subject`、`evidence` 和 `supportedFixes`。只修改被点名的
`pos`、`size`、`labelAt`、`labelDx`、`labelDy`、`route`、`fromSide` 或 `toSide`，每轮仅处理
当前诊断，不改 renderer，不降低到 `standard`。

- [ ] **Step 4: 交付 HTML**

Run:

```powershell
node "$env:USERPROFILE\.agents\skills\archify\bin\archify.mjs" deliver architecture `
  .\docs\marketing\xelora-ai-platform-panorama.architecture.json `
  .\docs\marketing\xelora-ai-platform-panorama.html `
  --quality showcase --json
```

Expected: `"ok": true`，并返回非空 `artifact.sha256` 和 `artifact.bytes`。

---

### Task 3: 创建“行业智能化应用生态图”

**Files:**
- Create: `docs/marketing/xelora-industry-ecosystem.architecture.json`

- [ ] **Step 1: 写入中心与四大产业集群**

使用以下 9 个组件，避免为 48 个场景创建 48 个节点：

```json
[
  {
    "id": "xelora_platform",
    "type": "backend",
    "label": "Xelora 企业级 AI 基盘",
    "sublabel": "知识 + 智能体 + Runtime",
    "tag": "跨行业复用",
    "pos": [610, 350],
    "size": [280, 96]
  },
  {
    "id": "industrial_cluster",
    "type": "cloud",
    "label": "工业与实体经济",
    "sublabel": "制造 / 汽车 / 能源 / 建筑",
    "pos": [230, 145],
    "size": [260, 82]
  },
  {
    "id": "finance_cluster",
    "type": "cloud",
    "label": "金融与现代服务",
    "sublabel": "银行 / 保险 / 零售 / 物流",
    "pos": [1010, 145],
    "size": [260, 82]
  },
  {
    "id": "public_cluster",
    "type": "cloud",
    "label": "公共与专业服务",
    "sublabel": "政务 / 法律 / 医疗 / 教育",
    "pos": [230, 575],
    "size": [260, 82]
  },
  {
    "id": "technology_cluster",
    "type": "cloud",
    "label": "科技与内容产业",
    "sublabel": "医药 / 通信 / 软件 / 文旅",
    "pos": [1010, 575],
    "size": [260, 82]
  },
  {
    "id": "knowledge_value",
    "type": "frontend",
    "label": "激活知识",
    "sublabel": "搜索 · 问答 · 共享",
    "pos": [610, 120],
    "size": [130, 64]
  },
  {
    "id": "decision_value",
    "type": "frontend",
    "label": "辅助决策",
    "sublabel": "洞察 · 分析 · 预警",
    "pos": [760, 120],
    "size": [130, 64]
  },
  {
    "id": "execution_value",
    "type": "messagebus",
    "label": "自动执行",
    "sublabel": "流程 · 工具 · 产物",
    "pos": [610, 690],
    "size": [130, 64]
  },
  {
    "id": "governance_value",
    "type": "security",
    "label": "安全可信",
    "sublabel": "权限 · 审批 · 审计",
    "pos": [760, 690],
    "size": [130, 64]
  }
]
```

从 `xelora_platform` 向四大产业集群建立四条 `dashed` 关系，标签分别为
`知识赋能`、`业务增效`、`专业辅助`、`创新加速`。价值节点不增加关系，通过空间邻近和
说明卡表达。

添加 5 张说明卡：

1. `工业与实体经济`：逐行业写入智能制造、汽车交通、能源电力、建筑地产的 12 个场景。
2. `金融与现代服务`：逐行业写入银行证券、保险、零售电商、物流供应链的 12 个场景。
3. `公共与专业服务`：逐行业写入政务、法律、医疗、教育的 12 个场景。
4. `科技与内容产业`：逐行业写入医药、通信、软件互联网、文旅传媒的 12 个场景。
5. `通用价值`：激活知识、辅助决策、自动执行、内容生产、风险合规、客户体验。

高风险场景文案必须保留：

```text
智能核保辅助
信贷材料审查辅助
合同审阅辅助
医学知识助手
病历摘要辅助
材料预审辅助
```

不得改写为“自动核保”“自动授信”“自动法律结论”“自动诊断”。

- [ ] **Step 2: 验证行业图候选**

Run:

```powershell
node "$env:USERPROFILE\.agents\skills\archify\bin\archify.mjs" validate architecture `
  .\docs\marketing\xelora-industry-ecosystem.architecture.json `
  --quality showcase --json
```

Expected: `"ok": true`、零 errors、零 warnings。

- [ ] **Step 3: 按诊断做最多两轮局部修复**

优先调整四条中心辐射关系的 `via`、`labelAt` 和节点 `pos`。不得通过删除行业场景卡、
降低质量配置或改动 Archify renderer 使验证通过。

- [ ] **Step 4: 交付行业生态 HTML**

Run:

```powershell
node "$env:USERPROFILE\.agents\skills\archify\bin\archify.mjs" deliver architecture `
  .\docs\marketing\xelora-industry-ecosystem.architecture.json `
  .\docs\marketing\xelora-industry-ecosystem.html `
  --quality showcase --json
```

Expected: `"ok": true`，并返回最终文件的 SHA-256 和字节数。

---

### Task 4: 生成 PNG 并执行像素级检查

**Files:**
- Create: `docs/marketing/xelora-ai-platform-panorama.png`
- Create: `docs/marketing/xelora-industry-ecosystem.png`

- [ ] **Step 1: 使用 Edge 生成 2560×1600 PNG**

Run:

```powershell
$edge = 'C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe'

& $edge --headless --disable-gpu --hide-scrollbars `
  --window-size=2560,1600 `
  --screenshot='E:\Xelora\WeKnora\docs\marketing\xelora-ai-platform-panorama.png' `
  'file:///E:/Xelora/WeKnora/docs/marketing/xelora-ai-platform-panorama.html'

& $edge --headless --disable-gpu --hide-scrollbars `
  --window-size=2560,1600 `
  --screenshot='E:\Xelora\WeKnora\docs\marketing\xelora-industry-ecosystem.png' `
  'file:///E:/Xelora/WeKnora/docs/marketing/xelora-industry-ecosystem.html'
```

Expected: 两个 PNG 均生成且文件大小大于 50 KB。

- [ ] **Step 2: 检查基盘全景图**

使用图片读取工具打开 `docs/marketing/xelora-ai-platform-panorama.png`，逐项确认：

```text
标题和副标题完整
六层主线清晰
九张说明卡无裁切
六大业务域均出现
核心能力/可选扩展/演进方向状态可区分
无文字重叠、组件重叠或边界穿线
```

- [ ] **Step 3: 检查行业生态图**

使用图片读取工具打开 `docs/marketing/xelora-industry-ecosystem.png`，逐项确认：

```text
中心 Xelora 节点是第一视觉焦点
四大产业集群空间分区明确
16 个行业均在说明卡中出现
48 个代表场景均可辨认
医疗、金融、法律场景保留“辅助”边界
无文字裁切、卡片溢出或误导性关系线
```

- [ ] **Step 4: 仅做视觉诊断对应的修复**

每张图最多两轮视觉修复。只修改诊断涉及的文案长度、`pos`、`size`、边距、关系路由或
viewBox；每次修改后重新运行 `validate` 和 `deliver`，然后重新截图检查。

---

### Task 5: 最终验收

**Files:**
- Verify: `docs/marketing/xelora-ai-platform-panorama.architecture.json`
- Verify: `docs/marketing/xelora-ai-platform-panorama.html`
- Verify: `docs/marketing/xelora-ai-platform-panorama.png`
- Verify: `docs/marketing/xelora-industry-ecosystem.architecture.json`
- Verify: `docs/marketing/xelora-industry-ecosystem.html`
- Verify: `docs/marketing/xelora-industry-ecosystem.png`

- [ ] **Step 1: 重跑确定性验证**

Run:

```powershell
node "$env:USERPROFILE\.agents\skills\archify\bin\archify.mjs" validate architecture `
  .\docs\marketing\xelora-ai-platform-panorama.architecture.json `
  --quality showcase --json

node "$env:USERPROFILE\.agents\skills\archify\bin\archify.mjs" validate architecture `
  .\docs\marketing\xelora-industry-ecosystem.architecture.json `
  --quality showcase --json
```

Expected: 两次均 `"ok": true`。

- [ ] **Step 2: 检查输出完整性**

Run:

```powershell
Get-Item .\docs\marketing\xelora-* |
  Select-Object Name, Length |
  Sort-Object Name
```

Expected: 六个文件存在，两个 HTML 和两个 PNG 均非空。

- [ ] **Step 3: 检查工作区状态**

Run:

```powershell
git status --short -- docs/marketing
```

Expected: 只出现本计划生成的六个宣传图文件。不要提交，除非用户另行明确要求。

- [ ] **Step 4: 输出真实交付回执**

最终回复必须包含：

```text
validation: passed
visual_review: passed
correction_rounds: <两张图各自的实际次数>
```

并提供两个 HTML、两个 JSON 和两个 PNG 的可点击路径。
