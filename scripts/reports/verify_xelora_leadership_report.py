from __future__ import annotations

import argparse
import sys
from pathlib import Path

from openpyxl import load_workbook
from PIL import Image as PILImage, ImageDraw


EXPECTED_SHEETS = [
    "01_领导摘要",
    "02_业务问题与价值闭环",
    "03_总体架构",
    "04_核心能力与产品截图",
    "05_亮点与技术优势",
    "06_使用场景与效果",
    "07_当前进展与后续规划",
    "08_证据与资料来源",
]
ERROR_TOKENS = {"#REF!", "#DIV/0!", "#VALUE!", "#N/A", "#NAME?", "#NULL!", "#NUM!"}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Verify and render the Xelora leadership Excel report")
    parser.add_argument("--input", type=Path, required=True)
    parser.add_argument("--render-dir", type=Path, required=True)
    return parser.parse_args()


def assert_workbook_shape(path: Path) -> int:
    wb = load_workbook(path, data_only=False, keep_links=False)
    try:
        assert wb.sheetnames == EXPECTED_SHEETS, wb.sheetnames
        assert all(ws.sheet_view.showGridLines is False for ws in wb.worksheets)
        assert all(ws.page_setup.orientation == "landscape" for ws in wb.worksheets)
        formula_count = sum(
            1
            for ws in wb.worksheets
            for row in ws.iter_rows()
            for cell in row
            if isinstance(cell.value, str) and cell.value.startswith("=")
        )
        image_count = sum(len(ws._images) for ws in wb.worksheets)
        assert formula_count >= 3, formula_count
        assert image_count >= 7, image_count
        status_ws = wb["07_当前进展与后续规划"]
        assert status_ws["B7"].value == '=COUNTIF($C$15:$C$200,"已完成/已验证")'
        assert status_ws["B8"].value == '=COUNTIF($C$15:$C$200,"进行中/当前工作区")'
        assert status_ws["B9"].value == '=COUNTIF($C$15:$C$200,"待规划")'
        print(f"Structure OK: sheets={len(wb.sheetnames)}, formulas={formula_count}, images={image_count}")
        return formula_count
    finally:
        wb.close()


def find_error_tokens(path: Path) -> list[str]:
    wb = load_workbook(path, data_only=True, keep_links=False)
    errors: list[str] = []
    try:
        for ws in wb.worksheets:
            for row in ws.iter_rows():
                for cell in row:
                    if isinstance(cell.value, str) and cell.value in ERROR_TOKENS:
                        errors.append(f"{ws.title}!{cell.coordinate}={cell.value}")
        status_ws = wb["07_当前进展与后续规划"]
        for coordinate in ("B7", "B8", "B9"):
            assert status_ws[coordinate].value is not None, f"missing calculated value: {coordinate}"
    finally:
        wb.close()
    return errors


def recalculate_and_export(input_path: Path, pdf_path: Path) -> None:
    import win32com.client  # type: ignore

    excel = win32com.client.DispatchEx("Excel.Application")
    excel.Visible = False
    excel.DisplayAlerts = False
    workbook = None
    try:
        workbook = excel.Workbooks.Open(str(input_path.resolve()), UpdateLinks=0, ReadOnly=False)
        excel.Calculation = -4105
        excel.CalculateFullRebuild()
        workbook.Save()
        pdf_path.parent.mkdir(parents=True, exist_ok=True)
        workbook.ExportAsFixedFormat(0, str(pdf_path.resolve()))
    finally:
        if workbook is not None:
            workbook.Close(SaveChanges=False)
        excel.Quit()


def render_pdf(pdf_path: Path, render_dir: Path) -> list[Path]:
    import fitz  # type: ignore

    render_dir.mkdir(parents=True, exist_ok=True)
    page_paths: list[Path] = []
    document = fitz.open(pdf_path)
    try:
        matrix = fitz.Matrix(2.0, 2.0)
        for index, page in enumerate(document, 1):
            pixmap = page.get_pixmap(matrix=matrix, alpha=False)
            page_path = render_dir / f"page-{index:02d}.png"
            pixmap.save(str(page_path))
            page_paths.append(page_path)
    finally:
        document.close()
    if not page_paths:
        raise ValueError("PDF export produced no pages")
    make_contact_sheet(page_paths, render_dir / "contact-sheet.png")
    return page_paths


def make_contact_sheet(page_paths: list[Path], output: Path) -> None:
    thumbnails: list[PILImage.Image] = []
    for page_path in page_paths:
        with PILImage.open(page_path) as image:
            thumbnail = image.convert("RGB")
            thumbnail.thumbnail((520, 370))
            thumbnails.append(thumbnail.copy())
    columns = 2
    rows = (len(thumbnails) + columns - 1) // columns
    margin = 24
    label_height = 24
    cell_width = 540
    cell_height = 410
    sheet = PILImage.new("RGB", (columns * cell_width + margin, rows * cell_height + margin), "#EEF2F5")
    draw = ImageDraw.Draw(sheet)
    for index, thumbnail in enumerate(thumbnails):
        col = index % columns
        row = index // columns
        left = margin // 2 + col * cell_width
        top = margin // 2 + row * cell_height
        draw.rectangle((left, top, left + cell_width - margin, top + cell_height - margin), fill="white", outline="#D9E1E8")
        draw.text((left + 10, top + 8), f"page-{index + 1:02d}", fill="#17324D")
        image_left = left + (cell_width - thumbnail.width) // 2 - margin // 2
        image_top = top + label_height + (cell_height - label_height - thumbnail.height) // 2 - margin // 2
        sheet.paste(thumbnail, (image_left, image_top))
    sheet.save(output)


def main() -> None:
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    args = parse_args()
    input_path = args.input.resolve()
    render_dir = args.render_dir.resolve()
    if not input_path.is_file():
        raise FileNotFoundError(input_path)
    formula_count = assert_workbook_shape(input_path)
    pdf_path = render_dir / "Xelora-项目架构与应用价值汇报.pdf"
    recalculate_and_export(input_path, pdf_path)
    errors = find_error_tokens(input_path)
    if errors:
        raise AssertionError("Formula errors found:\n" + "\n".join(errors))
    pages = render_pdf(pdf_path, render_dir)
    print(f"Formula check OK: formulas={formula_count}, errors=0")
    print(f"Rendered pages: {len(pages)}")
    print(f"PDF: {pdf_path}")
    print(f"Contact sheet: {render_dir / 'contact-sheet.png'}")


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"Verification failed: {exc}", file=sys.stderr)
        raise
