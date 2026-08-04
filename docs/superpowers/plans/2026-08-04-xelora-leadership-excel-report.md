# Xelora Leadership Excel Report Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generate a leadership-ready, image-rich Excel workbook that explains Xelora's architecture, technical advantages, representative use cases, evidence-backed outcomes, and current delivery status without overstating unfinished work.

**Architecture:** A reproducible Python generator reads the repository's README, task board, specifications, architecture diagram, and product screenshots, then embeds the evidence into eight formatted workbook sheets. The workbook is delivered outside the Git source tree, while the generator and verifier remain under `scripts/reports/`; Excel COM is used for Windows formula recalculation and PDF rendering because the current machine has Microsoft Excel but no `soffice` executable on `PATH`.

**Tech Stack:** Python 3.14, openpyxl 3.1.5, Pillow 12.1.0, pywin32 Excel COM, PyMuPDF (`fitz`), and the existing `xlsx` skill's `recalc.py` as the first formula-check attempt.

---

## File Map

| File | Responsibility |
| --- | --- |
| `scripts/reports/build_xelora_leadership_report.py` | Source preflight, evidence extraction, workbook content, formatting, charts, image embedding, and output generation. |
| `scripts/reports/verify_xelora_leadership_report.py` | Workbook structure checks, Excel recalculation, formula/error checks, PDF export, and rendered contact-sheet generation. |
| `E:/Xelora/deliverables/Xelora-项目架构与应用价值汇报.xlsx` | Final user-facing workbook, outside the Git repository. |
| `E:/Xelora/deliverables/rendered/` | Temporary PDF/PNG render evidence used for visual QA. |

Read-only source inputs:

- `README_CN.md` and `README.md` for product positioning, capabilities, deployment, model/provider, retrieval, and security claims.
- `docs/customizations/TASKS.md` for completed and pending customization work.
- `specs/002-session-workspace-binding/spec.md` and `specs/001-opensandbox-baseline/spec.md` for workspace, artifact, execution, and provider-boundary evidence.
- `docs/superpowers/specs/2026-08-03-xelora-pi-workspace-execution-boundary-design.md` for the active Personal/Pi ownership boundary.
- `docs/xelora-current-architecture.png` for the current architecture visual.
- `docs/images/qa.png`, `agent-qa.png`, `wiki-browser.png`, `wiki-graph.png`, `settings.png`, and `langfuse.png` for product evidence images.

The generator must never modify these inputs or any existing source/runtime file.

## Task 1: Add the Evidence-Driven Report Generator

**Files:**
- Create: `scripts/reports/build_xelora_leadership_report.py`

- [ ] **Step 1: Create the source contract and command-line entry point**

Add a UTF-8 Python script with these imports, paths, and source maps. The script must accept an explicit output path so the delivery location is not coupled to the repository:

```python
from __future__ import annotations

import argparse
import re
from dataclasses import dataclass
from pathlib import Path

from openpyxl import Workbook
from openpyxl.chart import BarChart, Reference
from openpyxl.drawing.image import Image as XLImage
from openpyxl.styles import Alignment, Border, Font, PatternFill, Side
from openpyxl.utils import get_column_letter
from openpyxl.worksheet.dimensions import ColumnDimension
from PIL import Image as PILImage

ROOT = Path(__file__).resolve().parents[2]

SOURCE_FILES = {
    "README_CN": ROOT / "README_CN.md",
    "README": ROOT / "README.md",
    "TASKS": ROOT / "docs/customizations/TASKS.md",
    "WORKSPACE_SPEC": ROOT / "specs/002-session-workspace-binding/spec.md",
    "EXECUTION_SPEC": ROOT / "specs/001-opensandbox-baseline/spec.md",
    "PI_DESIGN": ROOT / "docs/superpowers/specs/2026-08-03-xelora-pi-workspace-execution-boundary-design.md",
}

IMAGE_FILES = {
    "current_architecture": ROOT / "docs/xelora-current-architecture.png",
    "qa": ROOT / "docs/images/qa.png",
    "agent_qa": ROOT / "docs/images/agent-qa.png",
    "wiki_browser": ROOT / "docs/images/wiki-browser.png",
    "wiki_graph": ROOT / "docs/images/wiki-graph.png",
    "settings": ROOT / "docs/images/settings.png",
    "langfuse": ROOT / "docs/images/langfuse.png",
}

SHEET_NAMES = [
    "01_领导摘要",
    "02_业务问题与价值闭环",
    "03_总体架构",
    "04_核心能力与产品截图",
    "05_亮点与技术优势",
    "06_使用场景与效果",
    "07_当前进展与后续规划",
    "08_证据与资料来源",
]

def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Build the Xelora leadership Excel report")
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--check-sources", action="store_true")
    return parser.parse_args()
```

`--check-sources` must validate every source and image path, verify that images open with Pillow, and exit with status 0 only when all inputs are readable. A missing input must raise a message naming the exact path.

- [ ] **Step 2: Add the evidence data model and current-worktree status parser**

Define a small immutable evidence model and parse only explicit task-board checkboxes. Preserve the original task text and source path so the workbook can show evidence rather than inferred metrics:

```python
@dataclass(frozen=True)
class Evidence:
    claim: str
    interpretation: str
    status: str
    source: str
    source_kind: str

@dataclass(frozen=True)
class TaskRow:
    task_id: str
    title: str
    status: str
    source: str

STATUS_DONE = "已完成/已验证"
STATUS_CURRENT = "进行中/当前工作区"
STATUS_PLANNED = "待规划"

CHECKBOX_RE = re.compile(r"^\s*-\s*\[([ xX])\]\s+(.*)$")
TASK_ID_RE = re.compile(r"\bT-\d+\b")

def parse_task_board(path: Path) -> list[TaskRow]:
    rows: list[TaskRow] = []
    for raw_line in path.read_text(encoding="utf-8").splitlines():
        match = CHECKBOX_RE.match(raw_line)
        if not match:
            continue
        task_id = TASK_ID_RE.search(match.group(2))
        if not task_id:
            continue
        rows.append(TaskRow(
            task_id=task_id.group(0),
            title=match.group(2).strip(),
            status=STATUS_DONE if match.group(1).lower() == "x" else STATUS_PLANNED,
            source="docs/customizations/TASKS.md",
        ))
    return rows
```

Add one explicit current-worktree evidence row for the active Personal/Pi workspace execution boundary, sourced from the design specification, and add one row for the current working-tree SSE completion hardening only when the generator observes the corresponding files. Label both rows as `进行中/当前工作区`; do not convert the Git diff into a release claim.

- [ ] **Step 3: Add the report theme and common layout helpers**

Use one professional Chinese font (`Microsoft YaHei`) and the following stable palette. Keep the workbook white and grid-aligned, with cards only for repeated evidence items:

```python
THEME = {
    "navy": "17324D",
    "navy_2": "244A68",
    "green": "2F8061",
    "purple": "6558C8",
    "amber": "C98A24",
    "ink": "263445",
    "muted": "65758B",
    "line": "D9E1E8",
    "panel": "F5F7FA",
    "white": "FFFFFF",
}

def configure_sheet(ws, title_rows: str = "$1:$5") -> None:
    ws.sheet_view.showGridLines = False
    ws.sheet_properties.pageSetUpPr.fitToPage = True
    ws.page_setup.orientation = "landscape"
    ws.page_setup.paperSize = ws.PAPERSIZE_A4
    ws.page_setup.fitToWidth = 1
    ws.page_setup.fitToHeight = 0
    ws.page_margins.left = 0.25
    ws.page_margins.right = 0.25
    ws.page_margins.top = 0.45
    ws.page_margins.bottom = 0.45
    ws.print_title_rows = title_rows

def add_section_title(ws, row: int, title: str, subtitle: str = "") -> None:
    ws.merge_cells(start_row=row, start_column=1, end_row=row, end_column=12)
    cell = ws.cell(row=row, column=1, value=title)
    cell.font = Font(name="Microsoft YaHei", size=16, bold=True, color=THEME["navy"])
    cell.alignment = Alignment(vertical="center")
    ws.row_dimensions[row].height = 28
    if subtitle:
        ws.merge_cells(start_row=row + 1, start_column=1, end_row=row + 1, end_column=12)
        ws.cell(row=row + 1, column=1, value=subtitle).font = Font(
            name="Microsoft YaHei", size=9, color=THEME["muted"]
        )

def add_source_note(ws, row: int, text: str) -> None:
    ws.merge_cells(start_row=row, start_column=1, end_row=row, end_column=12)
    cell = ws.cell(row=row, column=1, value=f"来源：{text}")
    cell.font = Font(name="Microsoft YaHei", size=8, italic=True, color=THEME["muted"])
    cell.alignment = Alignment(wrap_text=True, vertical="top")
```

Add helpers for wrapping text, borders, colored status labels, consistent column widths, print areas, and proportional image sizing. Every merged title, image, and caption must have a deterministic anchor row; no automatic row-height logic may be used for image-bearing rows.

- [ ] **Step 4: Run the source preflight before adding workbook content**

Run:

```powershell
python scripts/reports/build_xelora_leadership_report.py --output 'E:\Xelora\deliverables\Xelora-项目架构与应用价值汇报.xlsx' --check-sources
```

Expected result: the command prints a successful source-check message and does not create or overwrite the workbook. Confirm that `README_CN.md`, the task/spec files, the current architecture PNG, and all six README screenshots are present.

- [ ] **Step 5: Commit only the generator source**

```powershell
git add -- scripts/reports/build_xelora_leadership_report.py
git diff --cached --check
git commit --only -m "feat: add Xelora leadership report generator" -- scripts/reports/build_xelora_leadership_report.py
```

## Task 2: Build the Eight Workbook Sheets

**Files:**
- Modify: `scripts/reports/build_xelora_leadership_report.py`

- [ ] **Step 1: Create the workbook shell and calculation settings**

Create the sheets in the exact order from `SHEET_NAMES`, remove the default sheet, apply `configure_sheet`, and force Excel to recalculate formulas when opened:

```python
wb = Workbook()
default = wb.active
wb.remove(default)
for name in SHEET_NAMES:
    configure_sheet(wb.create_sheet(name))
wb.calculation.calcMode = "auto"
wb.calculation.fullCalcOnLoad = True
wb.calculation.forceFullCalc = True
```

Set document properties to title `Xelora 项目架构与应用价值汇报`, subject `架构设计、亮点优势、使用场景与当前进展`, and creator `Xelora report generator`.

- [ ] **Step 2: Build `01_领导摘要` and `02_业务问题与价值闭环`**

`01_领导摘要` must contain:

- A dark navy title band with `Xelora 项目架构与应用价值汇报` and date `2026-08-04`.
- The positioning sentence: `面向企业知识资产的模块化 RAG + Agent 平台，让分散资料可检索、可推理、可沉淀、可治理。`
- Three capability pillars labeled `RAG 检索问答`, `ReAct Agent 多步编排`, and `Wiki 知识沉淀`.
- Four evidence cards labeled `模块化替换`, `企业治理`, `工作区与产物`, and `可观测运行`.
- `qa.png` as the dominant product image and `xelora-current-architecture.png` as a smaller architecture preview.
- A visible legend: `已验证事实 / 当前工作区 / 设计决策 / 路线图`.

`02_业务问题与价值闭环` must use a five-column table with `业务问题`, `Xelora 处理`, `可复用资产`, `领导视角价值`, and `证据来源`, followed by a four-step value chain row: `理解` -> `检索` -> `推理` -> `沉淀`. Use leadership language first and put BM25, dense retrieval, GraphRAG, MCP, and Langfuse in supporting notes.

- [ ] **Step 3: Build `03_总体架构` with ownership boundaries**

Embed `docs/xelora-current-architecture.png` at a fixed width that leaves readable captions and a right-side ownership table with these rows:

| Boundary | Xelora owns | Replaceable/external |
| --- | --- | --- |
| Conversation and answer | session, agent loop, SSE, citations, history | model provider |
| Knowledge | ingestion, indexing strategy, retrieval orchestration | vector DB, parser, embedding/rerank model |
| Execution | workspace binding, path authority, artifact routing, policy | Docker/provider/Pi sidecar implementation |
| Governance | tenant/RBAC, sharing, credential policy, audit contract | identity/provider deployment mechanism |

Add a callout below the table: `Personal/Pi workspace execution boundary：当前设计/工作流，不作为已完成的企业生产能力宣称。` Cite `docs/superpowers/specs/2026-08-03-xelora-pi-workspace-execution-boundary-design.md`.

- [ ] **Step 4: Build `04_核心能力与产品截图` as a captioned screenshot grid**

Create a two-column, three-row screenshot grid. Each block must include an embedded image, a 1-2 line caption, an evidence-type label, and a repository source path:

| Image | Caption | Evidence label |
| --- | --- | --- |
| `qa.png` | 知识库问答主流程 | README 产品截图 |
| `agent-qa.png` | 多步 Agent 与工具调用演示 | README/demo 截图，不作为通用性能基准 |
| `wiki-browser.png` | Wiki 导航与 Markdown 知识资产 | README 产品截图 |
| `wiki-graph.png` | Wiki 图谱关联探索 | README 产品截图 |
| `settings.png` | 模型、解析、存储与平台配置 | README 产品截图 |
| `langfuse.png` | Agent/工具链路追踪 | README 产品截图 |

Captions must remain outside the image bounds and must wrap without truncation.

- [ ] **Step 5: Build `05_亮点与技术优势` as an evidence matrix**

Use columns `优势维度`, `能力事实`, `对项目的意义`, `成熟度口径`, and `证据来源`. Add rows for multi-source/multi-format ingestion; BM25/dense/graph/rerank/pgvector HNSW retrieval; ReAct/MCP/network search; replaceable model/vector/object/deployment layers; tenant RBAC/resource sharing/audit; credential encryption/TLS/SSRF/Skill isolation; and Docker/Kubernetes/private/offline deployment.

Use maturity labels only as `README 已声明`, `规格已定义`, `任务板已记录`, or `当前工作区证据`. Do not add numeric scores, percentages, or unsupported rankings.

- [ ] **Step 6: Build `06_使用场景与效果` as a scenario-to-evidence table**

Create six scenario blocks using the exact columns `使用场景`, `用户问题`, `Xelora 流程`, `可见结果`, and `证据`. Populate them with enterprise knowledge Q&A, complex research/multi-step reasoning, Wiki/knowledge graph generation, workspace-scoped Markdown/spreadsheet/report/presentation output, private tenant-governed deployment, and multi-channel access. Use `qa.png`, `agent-qa.png`, `wiki-browser.png`, `wiki-graph.png`, `settings.png`, and the workspace specifications as the evidence basis.

Use qualitative wording such as `形成可追溯回答`, `输出可复用 Wiki 资产`, and `产物落入绑定工作区`; do not claim ROI, accuracy, latency, adoption, or productivity percentages.

- [ ] **Step 7: Build `07_当前进展与后续规划` with formula-backed counts**

Write a status table starting at row 15 with columns `编号`, `事项`, `状态`, `证据来源`, and `备注`. Populate completed and pending rows from `parse_task_board`, then append the current-worktree evidence rows described in Task 1. Add the summary table in `A7:B9` using formulas that reference the status column:

```excel
B7 = COUNTIF($C$15:$C$200,"已完成/已验证")
B8 = COUNTIF($C$15:$C$200,"进行中/当前工作区")
B9 = COUNTIF($C$15:$C$200,"待规划")
```

Add a three-bar chart using `A7:B9`, with the category labels as the three status bands. Format the bands with green, amber, and muted purple fills. Add the explicit planned items from the task board: browser automation, runtime observability/audit, client data push, client identity-model correction, enterprise CRUD proxy, and server-side resource creation.

- [ ] **Step 8: Build `08_证据与资料来源` as the audit appendix**

Create a source register with columns `编号`, `来源文件`, `覆盖内容`, `使用位置`, and `口径说明`. Include README/README_CN, task board, workspace/execution specifications, Pi design, current architecture PNG, each screenshot, relevant commit `d0ae7ec7`, and a working-tree status note. Add the legend definitions:

| 标签 | Meaning |
| --- | --- |
| 已验证事实 | README/spec/task/test/source evidence supports the statement. |
| 当前工作区 | Visible in the current dirty workspace or active design, not a release claim. |
| 设计决策 | Ownership or boundary chosen by the current architecture/design. |
| 路线图 | Pending task-board or roadmap item. |

Add the report-generation timestamp and the sentence `本报告不替代项目验收，不将演示截图解释为性能基准。`

- [ ] **Step 9: Add workbook-wide print and image integrity checks**

For every sheet, set a print area covering only populated rows and columns A:L, repeat title rows, freeze the first content row where useful, and set a footer containing the sheet name and `Xelora | evidence-backed draft`. Save only after checking that every image path was embedded and no image-bearing row is hidden.

- [ ] **Step 10: Commit only the completed generator change**

```powershell
git add -- scripts/reports/build_xelora_leadership_report.py
git diff --cached --check
git commit --only -m "feat: generate Xelora leadership Excel report" -- scripts/reports/build_xelora_leadership_report.py
```

## Task 3: Generate the Deliverable Workbook

**Files:**
- Create: `E:/Xelora/deliverables/Xelora-项目架构与应用价值汇报.xlsx`

- [ ] **Step 1: Create the external delivery directory**

```powershell
New-Item -ItemType Directory -Force -LiteralPath 'E:\Xelora\deliverables' | Out-Null
New-Item -ItemType Directory -Force -LiteralPath 'E:\Xelora\deliverables\rendered' | Out-Null
```

- [ ] **Step 2: Run the generator**

```powershell
python scripts/reports/build_xelora_leadership_report.py --output 'E:\Xelora\deliverables\Xelora-项目架构与应用价值汇报.xlsx'
```

Expected result: the output exists, has exactly the eight names in `SHEET_NAMES` in order, and the generator prints the absolute workbook path plus the embedded image count.

- [ ] **Step 3: Run a structural smoke check before recalculation**

```powershell
python -c "from openpyxl import load_workbook; p=r'E:\Xelora\deliverables\Xelora-项目架构与应用价值汇报.xlsx'; w=load_workbook(p, data_only=False); print(w.sheetnames); print('formulas', sum(1 for s in w for row in s.iter_rows() for c in row if isinstance(c.value, str) and c.value.startswith('='))); print('images', sum(len(s._images) for s in w)); w.close()"
```

Expected result: the sheet list matches the eight-sheet contract, the formula count is greater than zero because of the status dashboard, and the image count includes the architecture plus six product screenshots.

## Task 4: Add Verification and Rendering

**Files:**
- Create: `scripts/reports/verify_xelora_leadership_report.py`

- [ ] **Step 1: Add workbook structure and error checks**

Implement these exact checks in `main()`:

```python
EXPECTED_SHEETS = [
    "01_领导摘要", "02_业务问题与价值闭环", "03_总体架构",
    "04_核心能力与产品截图", "05_亮点与技术优势", "06_使用场景与效果",
    "07_当前进展与后续规划", "08_证据与资料来源",
]
ERROR_TOKENS = {"#REF!", "#DIV/0!", "#VALUE!", "#N/A", "#NAME?", "#NULL!", "#NUM!"}

def assert_workbook_shape(path: Path) -> int:
    wb = load_workbook(path, data_only=False)
    assert wb.sheetnames == EXPECTED_SHEETS
    assert all(ws.sheet_view.showGridLines is False for ws in wb.worksheets)
    formula_count = sum(
        1 for ws in wb.worksheets for row in ws.iter_rows()
        for cell in row if isinstance(cell.value, str) and cell.value.startswith("=")
    )
    assert formula_count >= 3
    assert sum(len(ws._images) for ws in wb.worksheets) >= 7
    wb.close()
    return formula_count

def find_error_tokens(path: Path) -> list[str]:
    wb = load_workbook(path, data_only=True)
    errors = []
    for ws in wb.worksheets:
        for row in ws.iter_rows():
            for cell in row:
                if isinstance(cell.value, str) and cell.value in ERROR_TOKENS:
                    errors.append(f"{ws.title}!{cell.coordinate}={cell.value}")
    wb.close()
    return errors
```

The script must fail with a non-zero exit code and print every failing cell. It must also report formula count, image count, and the populated row count for each sheet.

- [ ] **Step 2: Add Windows Excel recalculation and PDF export**

Use `win32com.client.DispatchEx("Excel.Application")` with `Visible = False`, `DisplayAlerts = False`, and a `try/finally` cleanup. Open the workbook read/write, set `Calculation = -4105` (automatic), call `CalculateFullRebuild()`, save, export the workbook with `ExportAsFixedFormat(0, pdf_path)`, close without saving again, and quit Excel. The output PDF must be `E:/Xelora/deliverables/rendered/Xelora-项目架构与应用价值汇报.pdf`.

Attempt the official skill checker first:

```powershell
python 'C:\Users\xuwh\.agents\skills\xlsx\scripts\recalc.py' 'E:\Xelora\deliverables\Xelora-项目架构与应用价值汇报.xlsx' 30
```

The current machine has no `soffice` executable in the standard LibreOffice locations, so if that command reports the expected LibreOffice-unavailable error, run the verifier's Excel COM path and record that Windows Excel supplied the recalculation evidence. Do not silently call the workbook verified if both paths fail.

- [ ] **Step 3: Render PDF pages to PNG contact sheets**

Use PyMuPDF to open the exported PDF, render each page at 144 DPI, save `page-01.png`, `page-02.png`, and so on under `E:/Xelora/deliverables/rendered/`, and create `contact-sheet.png` with a 2-column grid of thumbnails. The script must print the page count and every generated image path.

- [ ] **Step 4: Commit only the verifier**

```powershell
git add -- scripts/reports/verify_xelora_leadership_report.py
git diff --cached --check
git commit --only -m "test: validate Xelora leadership Excel report" -- scripts/reports/verify_xelora_leadership_report.py
```

## Task 5: Execute Validation and Visual QA

**Files:**
- Modify: `scripts/reports/build_xelora_leadership_report.py` if visual or formula defects are found.
- Modify: `scripts/reports/verify_xelora_leadership_report.py` if the verifier misses a required acceptance condition.
- Verify: `E:/Xelora/deliverables/Xelora-项目架构与应用价值汇报.xlsx`

- [ ] **Step 1: Run the complete verifier**

```powershell
python scripts/reports/verify_xelora_leadership_report.py --input 'E:\Xelora\deliverables\Xelora-项目架构与应用价值汇报.xlsx' --render-dir 'E:\Xelora\deliverables\rendered'
```

Expected result: exit code 0, all eight sheets present in order, no error tokens after recalculation, at least seven embedded images, and a PDF plus PNG render set.

- [ ] **Step 2: Inspect the rendered contact sheet and key pages**

Use the local image viewer on `E:/Xelora/deliverables/rendered/contact-sheet.png`, then inspect the pages for `01_领导摘要`, `03_总体架构`, `04_核心能力与产品截图`, `06_使用场景与效果`, and `07_当前进展与后续规划` individually. Check that:

- title bands and captions are readable at page width;
- architecture and screenshots are not clipped, stretched, or overlapped;
- long Chinese text wraps inside its cells;
- status colors and chart labels match the legend;
- the report clearly distinguishes verified, current-worktree, design, and planned statements;
- print areas contain no accidental blank columns or extra blank pages.

- [ ] **Step 3: Fix and rerun only when a concrete defect is observed**

For an image or layout defect, adjust the fixed image dimensions, row heights, column widths, or merged-cell boundaries in the generator, rebuild the workbook, and rerun the verifier. For a formula defect, inspect the exact formula coordinate printed by the verifier, correct the source range or formula, rebuild, recalculate, and rerun the error scan. Do not manually edit the generated workbook as the final fix.

- [ ] **Step 4: Perform the final worktree and deliverable audit**

```powershell
git diff --check
git status --short --branch
Get-Item -LiteralPath 'E:\Xelora\deliverables\Xelora-项目架构与应用价值汇报.xlsx' | Select-Object FullName,Length,LastWriteTime
```

Confirm that only the report generator/verifier commits belong to this work, existing user changes remain untouched, and the final workbook opens independently of the repository's current dirty state because all visuals are embedded.

## Plan Self-Review

1. **Spec coverage:** Tasks 1-2 cover the eight-sheet structure, leadership narrative, current architecture, screenshots, status bands, sources, and no-metric-overclaim rule. Tasks 3-5 cover external delivery, formula recalculation, rendering, and all acceptance checks.
2. **Placeholder scan:** Every command has a concrete path, every workbook sheet has named content, and no step depends on an unspecified image, metric, or future decision.
3. **Type consistency:** `Evidence`, `TaskRow`, `SOURCE_FILES`, `IMAGE_FILES`, and `SHEET_NAMES` are defined once and reused by the generator and verifier through the workbook contract.
4. **Scope boundary:** No product/runtime behavior changes are planned. The only repository additions are the report generator and verifier; the user-facing workbook and render evidence remain under `E:/Xelora/deliverables`.
