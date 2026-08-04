from __future__ import annotations

import argparse
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path

from openpyxl import Workbook
from openpyxl.chart import BarChart, Reference
from openpyxl.chart.label import DataLabelList
from openpyxl.drawing.image import Image as XLImage
from openpyxl.styles import Alignment, Border, Font, PatternFill, Side
from openpyxl.utils import get_column_letter
from openpyxl.worksheet.pagebreak import Break
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
FONT_NAME = "Microsoft YaHei"
STATUS_DONE = "已完成/已验证"
STATUS_CURRENT = "进行中/当前工作区"
STATUS_PLANNED = "待规划"
CHECKBOX_RE = re.compile(r"^\s*-\s*\[([ xX])\]\s+(.*)$")
TASK_ID_RE = re.compile(r"\bT-\d+\b")


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


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Build the Xelora leadership Excel report")
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--check-sources", action="store_true")
    return parser.parse_args()


def rel(path: Path) -> str:
    try:
        return path.relative_to(ROOT).as_posix()
    except ValueError:
        return path.as_posix()


def source_text(key: str) -> str:
    return SOURCE_FILES[key].read_text(encoding="utf-8")


def source_ref(key: str, needle: str, label: str | None = None) -> str:
    path = SOURCE_FILES[key]
    lines = source_text(key).splitlines()
    line_no = next((i for i, line in enumerate(lines, 1) if needle in line), None)
    prefix = label or needle
    if line_no is None:
        return f"{rel(path)} | {prefix}"
    return f"{rel(path)} | {prefix} | 约第 {line_no} 行"


def preflight_sources() -> None:
    missing = [str(path) for path in SOURCE_FILES.values() if not path.is_file()]
    missing.extend(str(path) for path in IMAGE_FILES.values() if not path.is_file())
    if missing:
        raise FileNotFoundError("Missing report inputs:\n" + "\n".join(missing))
    invalid_images: list[str] = []
    for path in IMAGE_FILES.values():
        try:
            with PILImage.open(path) as image:
                image.verify()
        except Exception as exc:
            invalid_images.append(f"{path}: {exc}")
    if invalid_images:
        raise ValueError("Unreadable report images:\n" + "\n".join(invalid_images))


def parse_task_board(path: Path) -> list[TaskRow]:
    rows: list[TaskRow] = []
    for raw_line in path.read_text(encoding="utf-8").splitlines():
        match = CHECKBOX_RE.match(raw_line)
        if not match:
            continue
        task_id = TASK_ID_RE.search(match.group(2))
        if not task_id:
            continue
        rows.append(
            TaskRow(
                task_id=task_id.group(0),
                title=match.group(2).strip(),
                status=STATUS_DONE if match.group(1).lower() == "x" else STATUS_PLANNED,
                source=rel(path),
            )
        )
    return rows


def current_worktree_rows() -> list[TaskRow]:
    target_files = {
        "internal/agent/finalize.go",
        "internal/event/event_data.go",
        "internal/handler/session/agent_stream_handler.go",
        "internal/handler/session/qa.go",
    }
    try:
        result = subprocess.run(
            ["git", "status", "--short"],
            cwd=ROOT,
            check=True,
            capture_output=True,
            text=True,
            encoding="utf-8",
        )
    except (OSError, subprocess.CalledProcessError):
        return []
    changed = {line[3:].strip() for line in result.stdout.splitlines() if len(line) >= 4}
    rows: list[TaskRow] = []
    if changed & target_files:
        rows.append(
            TaskRow(
                task_id="WORKTREE-001",
                title="Agent 完成语义与空答案处理正在当前工作区硬化",
                status=STATUS_CURRENT,
                source="当前工作区 git status + internal/handler/session/agent_stream_handler.go",
            )
        )
    pi_design = ROOT / "docs/superpowers/specs/2026-08-03-xelora-pi-workspace-execution-boundary-design.md"
    if pi_design.is_file():
        rows.append(
            TaskRow(
                task_id="WORKTREE-002",
                title="Personal/Pi workspace execution boundary处于设计与实现工作流",
                status=STATUS_CURRENT,
                source=rel(pi_design),
            )
        )
    return rows


def thin_side(color: str = THEME["line"]) -> Side:
    return Side(style="thin", color=color)


def cell_border(color: str = THEME["line"]) -> Border:
    side = thin_side(color)
    return Border(left=side, right=side, top=side, bottom=side)


def fill(color: str) -> PatternFill:
    return PatternFill(fill_type="solid", fgColor=color)


def font(size: int = 10, bold: bool = False, color: str = THEME["ink"], italic: bool = False) -> Font:
    return Font(name=FONT_NAME, size=size, bold=bold, color=color, italic=italic)


def set_value(ws, row: int, column: int, value, *, size: int = 10, bold: bool = False,
              color: str = THEME["ink"], background: str | None = None,
              horizontal: str = "left", vertical: str = "top", wrap: bool = True,
              border: Border | None = None, number_format: str | None = None) -> None:
    cell = ws.cell(row=row, column=column, value=value)
    cell.font = font(size=size, bold=bold, color=color)
    cell.alignment = Alignment(horizontal=horizontal, vertical=vertical, wrap_text=wrap)
    if background:
        cell.fill = fill(background)
    if border:
        cell.border = border
    if number_format:
        cell.number_format = number_format


def merge_value(ws, start_row: int, start_col: int, end_row: int, end_col: int, value,
                *, size: int = 10, bold: bool = False, color: str = THEME["ink"],
                background: str | None = None, horizontal: str = "left", vertical: str = "top",
                wrap: bool = True, border: Border | None = None) -> None:
    ws.merge_cells(start_row=start_row, start_column=start_col, end_row=end_row, end_column=end_col)
    set_value(
        ws, start_row, start_col, value, size=size, bold=bold, color=color,
        background=background, horizontal=horizontal, vertical=vertical, wrap=wrap, border=border,
    )
    if border:
        for row in ws.iter_rows(min_row=start_row, max_row=end_row, min_col=start_col, max_col=end_col):
            for cell in row:
                cell.border = border
                if background:
                    cell.fill = fill(background)


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
    ws.oddFooter.center.text = "Xelora | evidence-backed draft"
    ws.oddFooter.center.size = 8
    ws.oddFooter.center.font = FONT_NAME
    widths = [14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14]
    for index, width in enumerate(widths, 1):
        ws.column_dimensions[get_column_letter(index)].width = width


def add_header(ws, title: str, subtitle: str, *, date: str = "2026-08-04") -> None:
    merge_value(ws, 1, 1, 2, 12, title, size=20, bold=True, color=THEME["white"],
                background=THEME["navy"], vertical="center")
    ws.row_dimensions[1].height = 26
    ws.row_dimensions[2].height = 26
    merge_value(ws, 3, 1, 3, 9, subtitle, size=9, color=THEME["muted"], vertical="center")
    set_value(ws, 3, 10, "报告日期", size=9, bold=True, color=THEME["muted"], horizontal="right", vertical="center")
    merge_value(ws, 3, 11, 3, 12, date, size=9, color=THEME["muted"], horizontal="right", vertical="center")
    ws.row_dimensions[3].height = 20


def add_section_title(ws, row: int, title: str, subtitle: str = "") -> None:
    merge_value(ws, row, 1, row, 12, title, size=16, bold=True, color=THEME["navy"], vertical="center")
    ws.row_dimensions[row].height = 28
    if subtitle:
        merge_value(ws, row + 1, 1, row + 1, 12, subtitle, size=9, color=THEME["muted"], vertical="center")
        ws.row_dimensions[row + 1].height = 20


def add_source_note(ws, row: int, text: str) -> None:
    merge_value(ws, row, 1, row, 12, f"来源：{text}", size=8, color=THEME["muted"], vertical="top")
    ws.row_dimensions[row].height = 24


def style_table(ws, header_row: int, start_col: int, headers: list[str], rows: list[list[str]],
                *, row_height: int = 36, header_color: str = THEME["navy"] ) -> int:
    border = cell_border()
    for offset, header in enumerate(headers):
        set_value(ws, header_row, start_col + offset, header, size=9, bold=True,
                  color=THEME["white"], background=header_color, horizontal="center",
                  vertical="center", border=border)
    ws.row_dimensions[header_row].height = 28
    for row_offset, values in enumerate(rows, 1):
        row_no = header_row + row_offset
        for offset, value in enumerate(values):
            background = THEME["white"] if row_offset % 2 else THEME["panel"]
            set_value(ws, row_no, start_col + offset, value, size=9, background=background,
                      vertical="center", border=border)
        ws.row_dimensions[row_no].height = row_height
    return header_row + len(rows)


def style_grouped_table(ws, header_row: int, groups: list[tuple[int, int]], headers: list[str],
                        rows: list[list[str]], *, row_height: int = 40,
                        header_color: str = THEME["navy"]) -> int:
    border = cell_border()
    for (start_col, end_col), header in zip(groups, headers):
        merge_value(ws, header_row, start_col, header_row, end_col, header, size=9, bold=True,
                    color=THEME["white"], background=header_color, horizontal="center",
                    vertical="center", border=border)
    ws.row_dimensions[header_row].height = 28
    for row_offset, values in enumerate(rows, 1):
        row_no = header_row + row_offset
        background = THEME["white"] if row_offset % 2 else THEME["panel"]
        for (start_col, end_col), value in zip(groups, values):
            merge_value(ws, row_no, start_col, row_no, end_col, value, size=9,
                        color=THEME["ink"], background=background, vertical="center", border=border)
        ws.row_dimensions[row_no].height = row_height
    return header_row + len(rows)


def add_image(ws, path: Path, anchor: str, *, max_width: int, max_height: int) -> tuple[int, int]:
    with PILImage.open(path) as source:
        width, height = source.size
    scale = min(max_width / width, max_height / height)
    image = XLImage(str(path))
    image.width = max(1, int(width * scale))
    image.height = max(1, int(height * scale))
    ws.add_image(image, anchor)
    return image.width, image.height


def image_block(ws, path: Path, anchor_row: int, anchor_col: int, end_col: int,
                caption_row: int, caption: str, evidence: str, *, max_width: int = 440,
                max_height: int = 190) -> None:
    anchor = f"{get_column_letter(anchor_col)}{anchor_row}"
    add_image(ws, path, anchor, max_width=max_width, max_height=max_height)
    merge_value(ws, caption_row, anchor_col, caption_row, end_col, caption,
                size=10, bold=True, color=THEME["navy"], vertical="top")
    merge_value(ws, caption_row + 1, anchor_col, caption_row + 1, end_col, evidence,
                size=8, color=THEME["muted"], vertical="top")
    ws.row_dimensions[caption_row].height = 24
    ws.row_dimensions[caption_row + 1].height = 28


def add_card(ws, row: int, start_col: int, end_col: int, title: str, body: str, color: str) -> None:
    border = cell_border(color)
    merge_value(ws, row, start_col, row, end_col, title, size=10, bold=True,
                color=THEME["white"], background=color, vertical="center", border=border)
    merge_value(ws, row + 1, start_col, row + 2, end_col, body, size=9,
                color=THEME["ink"], background=THEME["panel"], vertical="center", border=border)
    ws.row_dimensions[row].height = 22
    ws.row_dimensions[row + 1].height = 24
    ws.row_dimensions[row + 2].height = 24


def add_legend(ws, row: int) -> None:
    labels = [
        ("已验证事实", THEME["green"]),
        ("当前工作区", THEME["amber"]),
        ("设计决策", THEME["purple"]),
        ("路线图", THEME["muted"]),
    ]
    col = 1
    for label, color in labels:
        merge_value(ws, row, col, row, col + 1, label, size=8, bold=True,
                    color=THEME["white"], background=color, horizontal="center", vertical="center")
        col += 3
    ws.row_dimensions[row].height = 20


def build_summary(wb: Workbook) -> None:
    ws = wb["01_领导摘要"]
    add_header(ws, "Xelora 项目架构与应用价值汇报", "模块化 RAG + Agent 平台 | 架构、亮点、场景与当前进展")
    merge_value(ws, 5, 1, 6, 12,
                "面向企业知识资产的模块化 RAG + Agent 平台，让分散资料可检索、可推理、可沉淀、可治理。",
                size=14, bold=True, color=THEME["navy"], background=THEME["panel"], vertical="center")
    ws.row_dimensions[5].height = 28
    ws.row_dimensions[6].height = 28
    add_section_title(ws, 8, "三项核心能力", "从知识接入到回答、推理和知识沉淀形成闭环")
    add_card(ws, 10, 1, 4, "RAG 检索问答", "面向企业文档的语义检索、关键词检索与多策略召回，形成可追溯回答。", THEME["green"])
    add_card(ws, 10, 5, 8, "ReAct Agent 多步编排", "将知识库、MCP 工具、网络搜索和任务步骤组合为可观察的多步流程。", THEME["purple"])
    add_card(ws, 10, 9, 12, "Wiki 知识沉淀", "把资料与问答结果组织为可浏览、可关联的 Markdown Wiki 和知识图谱。", THEME["amber"])
    add_section_title(ws, 15, "四项可汇报的项目亮点", "亮点以项目文档、任务板和现有产品截图为依据")
    add_card(ws, 17, 1, 3, "模块化替换", "模型、向量库、对象存储和部署形态可替换，降低单一供应商绑定。", THEME["navy_2"])
    add_card(ws, 17, 4, 6, "企业治理", "租户、角色、资源归属、组织共享和凭证保护形成治理边界。", THEME["green"])
    add_card(ws, 17, 7, 9, "工作区与产物", "会话绑定工作区，Agent 产物可路由到 Markdown、表格、报告和演示文稿文件。", THEME["purple"])
    add_card(ws, 17, 10, 12, "可观测运行", "Langfuse 和 Agent 工具链路提供过程追踪，便于定位和复盘。", THEME["amber"])
    add_section_title(ws, 22, "产品与架构证据", "一张产品界面图 + 一张当前架构图，帮助领导先看懂再追问")
    add_image(ws, IMAGE_FILES["qa"], "A24", max_width=610, max_height=260)
    add_image(ws, IMAGE_FILES["current_architecture"], "H24", max_width=470, max_height=260)
    merge_value(ws, 38, 1, 38, 7, "知识库问答界面：展示用户可直接感知的产品入口", size=9, color=THEME["navy"], vertical="top")
    merge_value(ws, 38, 8, 38, 12, "当前项目架构：展示 Xelora 拥有的契约与可替换基础设施", size=9, color=THEME["navy"], vertical="top")
    add_legend(ws, 40)
    add_source_note(ws, 42, f"{source_ref('README_CN', 'RAG', 'README_CN 核心能力章节')}；{rel(IMAGE_FILES['qa'])}；{rel(IMAGE_FILES['current_architecture'])}")
    ws.print_area = "A1:L42"
    ws.page_setup.fitToHeight = 1
    ws.freeze_panes = "A5"


def build_value_loop(wb: Workbook) -> None:
    ws = wb["02_业务问题与价值闭环"]
    add_header(ws, "业务问题与价值闭环", "用领导语言说明项目为什么值得投入，再用技术证据支撑判断")
    headers = ["业务问题", "Xelora 处理", "可复用资产", "领导视角价值", "证据来源"]
    rows = [
        ["资料分散、格式复杂", "统一接入 Feishu/Notion/Yuque 及 PDF、Word、图片、Excel 等资料", "统一知识库与索引", "降低信息查找和重复整理成本", source_ref("README_CN", "多源", "多源与多格式接入")],
        ["问题需要跨资料判断", "RAG 召回与 ReAct Agent 多步调用组合", "回答、步骤、引用、工具轨迹", "把一次问答升级为可复盘的研究过程", source_ref("README_CN", "ReAct", "ReAct Agent")],
        ["知识产出难以沉淀", "自动生成 Wiki Markdown 与关联图谱", "Wiki 页面、导航、图谱", "把一次解决方案变成团队可复用资产", source_ref("README_CN", "Wiki", "Wiki 模式")],
        ["执行过程和产物不透明", "会话绑定工作区，产物经过边界和路由契约", "工作区文件、结构化产物、追踪记录", "便于交付、审计与后续自动化", source_ref("WORKSPACE_SPEC", "workspace", "工作区绑定规格")],
    ]
    end_row = style_table(ws, 5, 1, headers, rows, row_height=58)
    add_section_title(ws, end_row + 3, "价值闭环", "从理解资料到形成可复用资产，价值在最后一步沉淀")
    steps = [
        ("01 理解", "解析文档结构、来源和上下文", THEME["navy_2"]),
        ("02 检索", "BM25、Dense、GraphRAG 等策略召回", THEME["green"]),
        ("03 推理", "Agent 组织多步检索、工具调用和回答", THEME["purple"]),
        ("04 沉淀", "Wiki、图谱、工作区文件与追踪记录", THEME["amber"]),
    ]
    step_layout = [(1, 2, 3), (4, 5, 6), (7, 8, 9), (10, 11, None)]
    for (title, body, color), (start_col, end_col, arrow_col) in zip(steps, step_layout):
        add_card(ws, end_row + 6, start_col, end_col, title, body, color)
        if arrow_col is not None:
            merge_value(ws, end_row + 7, arrow_col, end_row + 7, arrow_col, "→", size=20, bold=True,
                        color=THEME["muted"], horizontal="center", vertical="center")
    merge_value(ws, end_row + 11, 1, end_row + 12, 12,
                "技术说明：README 同时列出 BM25、Dense、GraphRAG、Parent-Child、Rerank、pgvector HNSW 等检索与索引能力；本报告将其作为能力选项，不解释为已完成的统一效果排名。",
                size=9, color=THEME["muted"], background=THEME["panel"], vertical="center")
    add_legend(ws, end_row + 15)
    add_source_note(ws, end_row + 17, f"{rel(SOURCE_FILES['README_CN'])}；{rel(SOURCE_FILES['WORKSPACE_SPEC'])}；{rel(SOURCE_FILES['PI_DESIGN'])}")
    ws.print_area = f"A1:L{end_row + 17}"
    ws.freeze_panes = "A5"


def build_architecture(wb: Workbook) -> None:
    ws = wb["03_总体架构"]
    add_header(ws, "总体架构", "模块化分层架构：Xelora 负责产品契约，基础设施和运行时可按部署形态替换")
    add_image(ws, IMAGE_FILES["current_architecture"], "A5", max_width=1130, max_height=410)
    merge_value(ws, 28, 1, 28, 12,
                "架构图重点：访问入口、应用核心、执行/产物链、数据/模型基础设施和治理带分别承担不同职责。",
                size=10, bold=True, color=THEME["navy"], vertical="center")
    headers = ["边界", "Xelora 拥有的契约", "可替换/外部实现"]
    rows = [
        ["会话与回答", "session、agent loop、SSE、引用、历史", "模型供应商"],
        ["知识能力", "接入、索引策略、检索编排", "向量库、解析器、embedding/rerank 模型"],
        ["执行与产物", "工作区绑定、路径权威、产物路由、策略", "Docker/provider/Pi sidecar 实现"],
        ["治理与安全", "租户/RBAC、共享、凭证策略、审计契约", "身份供应商与部署机制"],
    ]
    end_row = style_table(ws, 30, 1, headers, rows, row_height=42)
    merge_value(ws, end_row + 2, 1, end_row + 3, 12,
                "Personal/Pi workspace execution boundary：当前设计与实现工作流；Go Agent 仍负责智能编排、知识检索和最终回答，Pi 只承担受限工作区执行，不作为已完成的企业生产能力宣称。",
                size=10, bold=True, color=THEME["purple"], background="F0EEFF", vertical="center", border=cell_border(THEME["purple"]))
    add_legend(ws, end_row + 6)
    add_source_note(ws, end_row + 8, f"{rel(IMAGE_FILES['current_architecture'])}；{rel(SOURCE_FILES['PI_DESIGN'])}；{rel(SOURCE_FILES['EXECUTION_SPEC'])}")
    ws.print_area = f"A1:L{end_row + 8}"
    ws.page_setup.fitToHeight = 1
    ws.freeze_panes = "A5"


def build_screenshots(wb: Workbook) -> None:
    ws = wb["04_核心能力与产品截图"]
    add_header(ws, "核心能力与产品截图", "现有 README 截图作为产品证据；演示截图不等同于通用性能基准")
    image_block(ws, IMAGE_FILES["qa"], 5, 1, 6, 18, "知识库问答主流程", "README 产品截图 | docs/images/qa.png")
    image_block(ws, IMAGE_FILES["agent_qa"], 5, 7, 12, 18, "多步 Agent 与工具调用演示", "README/demo 截图，不作为通用性能基准 | docs/images/agent-qa.png")
    image_block(ws, IMAGE_FILES["wiki_browser"], 22, 1, 6, 35, "Wiki 导航与 Markdown 知识资产", "README 产品截图 | docs/images/wiki-browser.png")
    image_block(ws, IMAGE_FILES["wiki_graph"], 22, 7, 12, 35, "Wiki 图谱关联探索", "README 产品截图 | docs/images/wiki-graph.png")
    image_block(ws, IMAGE_FILES["settings"], 39, 1, 6, 52, "模型、解析、存储与平台配置", "README 产品截图 | docs/images/settings.png")
    image_block(ws, IMAGE_FILES["langfuse"], 39, 7, 12, 52, "Agent 与工具链路追踪", "README 产品截图 | docs/images/langfuse.png")
    add_legend(ws, 56)
    add_source_note(ws, 58, f"{rel(SOURCE_FILES['README_CN'])}；{', '.join(rel(path) for path in IMAGE_FILES.values() if path != IMAGE_FILES['current_architecture'])}")
    ws.print_area = "A1:L58"
    ws.row_breaks.append(Break(id=38))
    ws.page_setup.fitToHeight = 2
    ws.freeze_panes = "A5"


def build_advantages(wb: Workbook) -> None:
    ws = wb["05_亮点与技术优势"]
    add_header(ws, "亮点与技术优势", "只呈现有 README、规格、任务板或当前工作区证据支撑的能力")
    headers = ["优势维度", "能力事实", "对项目的意义", "成熟度口径", "证据来源"]
    rows = [
        ["知识接入", "Feishu/Notion/Yuque；PDF、Word、图片、Excel 等多源多格式", "适应企业既有资料体系", "README 已声明", source_ref("README_CN", "多源", "数据接入")],
        ["检索策略", "BM25、Dense、GraphRAG、Parent-Child、Rerank、pgvector HNSW", "支持按知识库特征选择召回路径", "README 已声明", source_ref("README_CN", "BM25", "检索策略")],
        ["Agent 编排", "ReAct、多步任务、MCP 工具、网络搜索、可配置 Prompt", "适合复杂研究和跨工具任务", "README 已声明", source_ref("README_CN", "MCP", "Agent 与工具")],
        ["模块化替换", "模型、向量数据库、对象存储、部署形态可替换", "便于私有化、成本和供应商策略调整", "README 已声明", source_ref("README_CN", "向量数据库", "基础设施兼容")],
        ["企业治理", "四级租户 RBAC、资源归属、组织共享、审计日志", "支撑多团队和企业知识边界", "README 已声明/规格已定义", source_ref("README_CN", "RBAC", "权限与治理")],
        ["安全边界", "AES-256-GCM、TLS/token 保护、SSRF-aware HTTP、Skill 隔离", "降低凭证、网络和执行风险", "README 已声明/规格已定义", source_ref("README_CN", "AES-256", "安全能力")],
        ["部署适配", "Docker Compose、Kubernetes/Helm、私有化与离线部署路径", "适配不同交付环境", "README 已声明", source_ref("README_CN", "Kubernetes", "部署方式")],
    ]
    end_row = style_table(ws, 5, 1, headers, rows, row_height=60)
    merge_value(ws, end_row + 2, 1, end_row + 3, 12,
                "口径提醒：README 中的模型供应商、向量库等数量是兼容范围描述，不是质量分数；截图和 demo 是功能可见性证据，不是准确率或延迟基准。",
                size=10, bold=True, color=THEME["amber"], background="FFF7E8", vertical="center", border=cell_border(THEME["amber"]))
    add_legend(ws, end_row + 6)
    add_source_note(ws, end_row + 8, f"{rel(SOURCE_FILES['README_CN'])}；{rel(SOURCE_FILES['EXECUTION_SPEC'])}")
    ws.print_area = f"A1:L{end_row + 8}"
    ws.freeze_panes = "A5"


def build_scenarios(wb: Workbook) -> None:
    ws = wb["06_使用场景与效果"]
    add_header(ws, "使用场景与效果", "场景 -> 用户问题 -> Xelora 流程 -> 可见结果 -> 证据")
    group_headers = [(1, 2, "使用场景"), (3, 4, "用户问题"), (5, 7, "Xelora 流程"), (8, 10, "可见结果"), (11, 12, "证据")]
    border = cell_border()
    for start, end, title in group_headers:
        merge_value(ws, 5, start, 5, end, title, size=9, bold=True, color=THEME["white"],
                    background=THEME["navy"], horizontal="center", vertical="center", border=border)
    ws.row_dimensions[5].height = 28
    scenarios = [
        ("企业知识问答", "资料分散，业务人员难以快速找到依据", "文档接入 -> 多策略检索 -> 生成带引用回答", "形成可追溯回答，减少重复翻找", "qa.png + README"),
        ("复杂研究与多步推理", "问题需要跨知识库、工具和网络信息判断", "ReAct Agent -> RAG/MCP/网络搜索 -> 逐步回答", "过程可观察、结果可复盘", "agent-qa.png + README"),
        ("Wiki 与知识图谱", "一次解决方案难以沉淀为团队资产", "资料理解 -> Wiki 页面 -> 导航与图谱关联", "形成可浏览、可关联的知识资产", "wiki-browser.png + wiki-graph.png"),
        ("工作区文件产出", "回答之外还需要报告、表格和演示文件", "会话绑定工作区 -> Agent 工具执行 -> 产物路由", "产物落入指定工作区，可继续编辑和交付", "workspace spec + 当前工作区"),
        ("私有化与组织治理", "知识、凭证和访问边界不能依赖公共服务", "租户/RBAC -> 凭证保护 -> 私有/离线部署", "在组织边界内复用知识能力", "README + execution spec"),
        ("多渠道访问", "用户在不同协作入口提出知识问题", "Web/API/IM/Chrome/小程序等入口 -> 统一 Agent 核心", "同一知识与治理能力覆盖更多入口", "README 渠道能力章节"),
    ]
    for row_offset, values in enumerate(scenarios, 6):
        background = THEME["white"] if row_offset % 2 else THEME["panel"]
        for start, end, value in zip((1, 3, 5, 8, 11), (2, 4, 7, 10, 12), values):
            merge_value(ws, row_offset, start, row_offset, end, value, size=9,
                        color=THEME["ink"], background=background, vertical="center", border=border)
        ws.row_dimensions[row_offset].height = 66
    add_section_title(ws, 14, "场景截图", "使用已存在的产品截图辅助说明，不将其解释为生产指标")
    add_image(ws, IMAGE_FILES["qa"], "A16", max_width=320, max_height=150)
    add_image(ws, IMAGE_FILES["agent_qa"], "E16", max_width=320, max_height=150)
    add_image(ws, IMAGE_FILES["wiki_graph"], "I16", max_width=320, max_height=150)
    merge_value(ws, 27, 1, 27, 4, "问答入口", size=9, bold=True, color=THEME["navy"], vertical="top")
    merge_value(ws, 27, 5, 27, 8, "多步 Agent 演示", size=9, bold=True, color=THEME["navy"], vertical="top")
    merge_value(ws, 27, 9, 27, 12, "知识图谱探索", size=9, bold=True, color=THEME["navy"], vertical="top")
    add_legend(ws, 29)
    add_source_note(ws, 31, f"{rel(SOURCE_FILES['README_CN'])}；{rel(SOURCE_FILES['WORKSPACE_SPEC'])}；{rel(SOURCE_FILES['EXECUTION_SPEC'])}")
    ws.print_area = "A1:L31"
    ws.row_breaks.append(Break(id=13))
    ws.page_setup.fitToHeight = 2
    ws.freeze_panes = "A6"


def build_status(wb: Workbook) -> None:
    ws = wb["07_当前进展与后续规划"]
    add_header(ws, "当前进展与后续规划", "以任务板和当前工作区为依据，区分已验证、进行中和待规划")
    for column, width in {"A": 16, "B": 42, "C": 20, "D": 30, "E": 22}.items():
        ws.column_dimensions[column].width = width
    merge_value(ws, 5, 1, 5, 12, "状态总览", size=12, bold=True, color=THEME["navy"], vertical="center")
    status_rows = [("已完成", STATUS_DONE, THEME["green"]), ("进行中", STATUS_CURRENT, THEME["amber"]), ("待规划", STATUS_PLANNED, THEME["purple"])]
    for row, (label, formula_status, color) in enumerate(status_rows, 7):
        set_value(ws, row, 1, label, size=10, bold=True, color=THEME["white"], background=color,
                  horizontal="center", vertical="center", border=cell_border(color))
        set_value(ws, row, 2, f'=COUNTIF($C$15:$C$200,"{formula_status}")', size=12, bold=True,
                  color=THEME["ink"], background=THEME["panel"], horizontal="center", vertical="center",
                  border=cell_border(), number_format="0")
        ws.row_dimensions[row].height = 24
    chart = BarChart()
    chart.type = "bar"
    chart.style = 10
    chart.title = "事项状态分布"
    chart.y_axis.title = None
    chart.x_axis.title = None
    chart.height = 5.6
    chart.width = 10.0
    data = Reference(ws, min_col=2, min_row=7, max_row=9)
    categories = Reference(ws, min_col=1, min_row=7, max_row=9)
    chart.add_data(data, titles_from_data=False)
    chart.set_categories(categories)
    chart.dataLabels = DataLabelList()
    chart.dataLabels.showVal = True
    chart.dataLabels.showSerName = False
    chart.dataLabels.showCatName = True
    chart.dataLabels.showLegendKey = False
    chart.legend = None
    ws.add_chart(chart, "G6")
    tasks = parse_task_board(SOURCE_FILES["TASKS"]) + current_worktree_rows()
    headers = ["编号", "事项", "状态", "证据来源", "备注"]
    rows: list[list[str]] = []
    for task in tasks:
        title = re.sub(rf"^{re.escape(task.task_id)}\s*", "", task.title).replace(" - ", " / ")
        if len(title) > 52:
            title = title[:49] + "..."
        source = "TASKS.md" if task.source.endswith("docs/customizations/TASKS.md") else task.source
        source = source.replace("docs/superpowers/specs/", "specs/")
        note = "任务板已勾选" if task.status == STATUS_DONE else "未勾选/工作区证据"
        rows.append([task.task_id, title, task.status, source, note])
    end_row = style_table(ws, 14, 1, headers, rows, row_height=34)
    if end_row < 15:
        raise ValueError("Task board did not produce any status rows")
    merge_value(ws, end_row + 2, 1, end_row + 3, 12,
                "后续规划重点：浏览器自动化、运行时可观测与审计、客户端数据推送、客户端身份模型修正、企业资源 CRUD 代理、服务端知识库/Agent 创建。",
                size=10, bold=True, color=THEME["muted"], background=THEME["panel"], vertical="center", border=cell_border())
    add_legend(ws, end_row + 6)
    add_source_note(ws, end_row + 8, f"{rel(SOURCE_FILES['TASKS'])}；当前工作区 git status；{rel(SOURCE_FILES['PI_DESIGN'])}")
    ws.print_area = f"A1:L{end_row + 8}"
    ws.print_title_rows = "$1:$14"
    ws.page_setup.fitToHeight = 2
    ws.freeze_panes = "A15"


def build_sources(wb: Workbook) -> None:
    ws = wb["08_证据与资料来源"]
    add_header(ws, "证据与资料来源", "每个主要判断都保留来源边界，方便领导追问时回到原始材料")
    headers = ["编号", "来源文件", "覆盖内容", "使用位置", "口径说明"]
    rows = [
        ["S-01", rel(SOURCE_FILES["README_CN"]), "产品定位、RAG、Agent、Wiki、接入、检索、治理、部署", "01-06", "README 已声明能力"],
        ["S-02", rel(SOURCE_FILES["README"]), "英文功能与架构补充", "05、08", "与中文 README 交叉核对"],
        ["S-03", rel(SOURCE_FILES["TASKS"]), "完成与待规划 customization 事项", "07", "任务板状态，不替代验收报告"],
        ["S-04", rel(SOURCE_FILES["WORKSPACE_SPEC"]), "会话绑定、路径边界、产物路由、文件能力桥", "02、03、06", "规格与验证记录"],
        ["S-05", rel(SOURCE_FILES["EXECUTION_SPEC"]), "可替换执行 provider、受控 Docker 与治理边界", "03、05、06", "规格中的架构边界"],
        ["S-06", rel(SOURCE_FILES["PI_DESIGN"]), "Personal/Pi workspace ownership boundary", "03、07", "当前设计/工作流，不是发布声明"],
        ["S-07", rel(IMAGE_FILES["current_architecture"]), "当前 Xelora 分层架构图", "01、03", "当前工作区架构资产"],
        ["S-08", "docs/images/*.png", "QA、Agent、Wiki、Settings、Langfuse 产品截图", "01、04、06", "产品/演示截图，不是性能基准"],
        ["S-09", "git commit d0ae7ec7", "本报告设计稿", "08", "设计已单独提交"],
        ["S-10", "当前工作区 git status", "Agent 完成语义和 Pi workspace 当前变更", "07", "未提交工作区证据，不能等同发布能力"],
    ]
    source_groups = [(1, 2), (3, 5), (6, 8), (9, 10), (11, 12)]
    end_row = style_grouped_table(ws, 5, source_groups, headers, rows, row_height=42)
    add_section_title(ws, end_row + 3, "口径图例", "报告中的事实标签用于帮助非研发读者区分证据强度")
    legend_rows = [
        ["已验证事实", "README/规格/任务/源代码/测试支持的陈述", "绿色"],
        ["当前工作区", "dirty workspace 或 active design，不是发布声明", "琥珀色"],
        ["设计决策", "架构/规格明确的 ownership 与边界", "紫色"],
        ["路线图", "任务板/ROADMAP 中尚未完成的方向", "灰紫色"],
    ]
    legend_groups = [(1, 2), (3, 9), (10, 12)]
    style_grouped_table(ws, end_row + 5, legend_groups, ["标签", "含义", "视觉标识"], legend_rows,
                        row_height=30, header_color=THEME["navy_2"])
    merge_value(ws, end_row + 11, 1, end_row + 12, 12,
                "本报告不替代项目验收，不将演示截图解释为性能基准；任何 ROI、准确率、延迟、采用率和生产规模结论都需要额外的独立测量证据。",
                size=10, bold=True, color=THEME["amber"], background="FFF7E8", vertical="center", border=cell_border(THEME["amber"]))
    add_source_note(ws, end_row + 15, "报告生成器读取以上来源并将图片嵌入 XLSX；最终工作簿可脱离源仓库打开。")
    ws.print_area = f"A1:L{end_row + 15}"
    ws.page_setup.fitToHeight = 1
    ws.freeze_panes = "A5"


def finalize_sheet(ws) -> None:
    for row in ws.iter_rows():
        for cell in row:
            if cell.value is not None and cell.font.name != FONT_NAME:
                cell.font = Font(name=FONT_NAME, size=cell.font.sz or 10, bold=cell.font.bold,
                                 italic=cell.font.italic, color=cell.font.color or THEME["ink"])
    ws.sheet_properties.pageSetUpPr.fitToPage = True


def assert_no_overlapping_merges(wb: Workbook) -> None:
    for ws in wb.worksheets:
        ranges = list(ws.merged_cells.ranges)
        for index, first in enumerate(ranges):
            for second in ranges[index + 1:]:
                rows_overlap = first.min_row <= second.max_row and second.min_row <= first.max_row
                cols_overlap = first.min_col <= second.max_col and second.min_col <= first.max_col
                if rows_overlap and cols_overlap:
                    raise ValueError(f"Overlapping merged ranges in {ws.title}: {first} and {second}")


def build_workbook(output: Path) -> None:
    preflight_sources()
    output.parent.mkdir(parents=True, exist_ok=True)
    wb = Workbook()
    default = wb.active
    wb.remove(default)
    for name in SHEET_NAMES:
        configure_sheet(wb.create_sheet(name))
    wb.calculation.calcMode = "auto"
    wb.calculation.fullCalcOnLoad = True
    wb.calculation.forceFullCalc = True
    wb.properties.title = "Xelora 项目架构与应用价值汇报"
    wb.properties.subject = "架构设计、亮点优势、使用场景与当前进展"
    wb.properties.creator = "Xelora report generator"
    build_summary(wb)
    build_value_loop(wb)
    build_architecture(wb)
    build_screenshots(wb)
    build_advantages(wb)
    build_scenarios(wb)
    build_status(wb)
    build_sources(wb)
    assert_no_overlapping_merges(wb)
    for ws in wb.worksheets:
        finalize_sheet(ws)
    wb.save(output)
    image_count = sum(len(ws._images) for ws in wb.worksheets)
    formula_count = sum(
        1 for ws in wb.worksheets for row in ws.iter_rows()
        for cell in row if isinstance(cell.value, str) and cell.value.startswith("=")
    )
    print(f"Created: {output.resolve()}")
    print(f"Sheets: {', '.join(wb.sheetnames)}")
    print(f"Embedded images: {image_count}")
    print(f"Formula cells: {formula_count}")


def main() -> None:
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    args = parse_args()
    preflight_sources()
    if args.check_sources:
        print(f"Source check passed: {len(SOURCE_FILES)} text sources, {len(IMAGE_FILES)} images")
        return
    build_workbook(args.output)


if __name__ == "__main__":
    main()
