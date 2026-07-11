#!/usr/bin/env python3
import json
import os
import shutil
import subprocess
import sys
import tempfile
import uuid
from pathlib import Path


MUTATING_ACTIONS = {
    "create",
    "save",
    "remove",
    "set",
    "add",
    "batch",
    "write_docx",
    "write_xlsx",
}


def resolve_relative_path(candidate):
    if candidate is None:
        raise ValueError("path is required")
    normalized = os.path.normpath(str(candidate).strip())
    if normalized in ("", ".", os.sep):
        raise ValueError(f"invalid relative path: {candidate}")
    if os.path.isabs(normalized) or normalized.startswith(".."):
        raise ValueError(f"invalid relative path: {candidate}")
    return normalized.replace("\\", "/")


def props_to_cli_args(props):
    if not props:
        return []
    args = []
    for key in sorted(props.keys()):
        value = props[key]
        args.extend(["--prop", f"{key}={value}"])
    return args


def require_field(payload, name):
    value = payload.get(name)
    if value in (None, ""):
        raise ValueError(f"{name} is required")
    return value


def build_officecli_command(payload, request_path):
    action = require_field(payload, "action")
    if action == "create":
        file_path = resolve_relative_path(require_field(payload, "file"))
        cmd = ["officecli", "create", file_path, "--json"]
        if payload.get("force"):
            cmd.append("--force")
        if payload.get("type"):
            cmd.extend(["--type", str(payload["type"])])
        return cmd, None

    file_path = resolve_relative_path(require_field(payload, "file"))

    if action == "validate":
        return ["officecli", "validate", file_path, "--json"], None
    if action == "save":
        return ["officecli", "save", file_path, "--json"], None
    if action == "close":
        return ["officecli", "close", file_path, "--json"], None
    if action == "view":
        mode = require_field(payload, "mode")
        cmd = ["officecli", "view", file_path, str(mode)]
        if payload.get("page") is not None:
            cmd.extend(["--page", str(payload["page"])])
        if payload.get("range") is not None:
            cmd.extend(["--range", str(payload["range"])])
        if payload.get("max_lines") is not None:
            cmd.extend(["--max-lines", str(payload["max_lines"])])
        if payload.get("out") is not None:
            cmd.extend(["--out", resolve_relative_path(payload["out"])])
        cmd.append("--json")
        return cmd, None
    if action == "get":
        path = payload.get("path", "/")
        cmd = ["officecli", "get", file_path, str(path), "--json"]
        if payload.get("depth") is not None:
            cmd.extend(["--depth", str(payload["depth"])])
        return cmd, None
    if action == "query":
        selector = require_field(payload, "selector")
        cmd = ["officecli", "query", file_path, str(selector), "--json"]
        if payload.get("find"):
            cmd.extend(["--find", str(payload["find"])])
        return cmd, None
    if action == "remove":
        path = require_field(payload, "path")
        return ["officecli", "remove", file_path, str(path), "--json"], None
    if action == "set":
        path = require_field(payload, "path")
        cmd = ["officecli", "set", file_path, str(path), "--json"]
        cmd.extend(props_to_cli_args(payload.get("props")))
        if payload.get("find"):
            cmd.extend(["--find", str(payload["find"])])
        if payload.get("replace") is not None:
            cmd.extend(["--replace", str(payload["replace"])])
        return cmd, None
    if action == "add":
        parent = require_field(payload, "parent")
        cmd = ["officecli", "add", file_path, str(parent), "--json"]
        if payload.get("type"):
            cmd.extend(["--type", str(payload["type"])])
        if payload.get("from"):
            cmd.extend(["--from", str(payload["from"])])
        if payload.get("index") is not None:
            cmd.extend(["--index", str(payload["index"])])
        if payload.get("after"):
            cmd.extend(["--after", str(payload["after"])])
        if payload.get("before"):
            cmd.extend(["--before", str(payload["before"])])
        cmd.extend(props_to_cli_args(payload.get("props")))
        return cmd, None
    if action == "batch":
        commands = payload.get("commands")
        if not isinstance(commands, list) or not commands:
            raise ValueError("commands must be a non-empty array for batch")
        fd, temp_path = tempfile.mkstemp(
            prefix="officecli-batch-",
            suffix=".json",
            dir=os.path.dirname(os.path.abspath(request_path)),
        )
        os.close(fd)
        with open(temp_path, "w", encoding="utf-8") as handle:
            json.dump(commands, handle, ensure_ascii=False)
        cmd = ["officecli", "batch", file_path, "--input", temp_path, "--json"]
        if payload.get("stop_on_error"):
            cmd.append("--stop-on-error")
        return cmd, temp_path

    raise ValueError(f"unsupported action: {action}")


def resolve_workspace_path(base_dir, candidate):
    relative = resolve_relative_path(candidate)
    resolved = (Path(base_dir) / relative).resolve()
    try:
        resolved.relative_to(Path(base_dir).resolve())
    except ValueError as exc:
        raise ValueError(f"path escapes conversation workspace: {candidate}") from exc
    return resolved


def print_completed(completed, replacements=None):
    replacements = replacements or {}

    def display(value):
        for source, target in replacements.items():
            value = value.replace(source, target)
        return value

    if completed.stdout:
        print(display(completed.stdout), end="")
    if completed.stderr:
        print(display(completed.stderr), file=sys.stderr, end="")


def run_officecli(command):
    env = os.environ.copy()
    env["LANG"] = "C.UTF-8"
    env["LC_ALL"] = "C.UTF-8"
    return subprocess.run(
        command,
        capture_output=True,
        text=True,
        check=False,
        env=env,
    )


def run_read_only(payload, request_path):
    command, temp_path = build_officecli_command(payload, str(request_path))
    try:
        completed = run_officecli(command)
        print_completed(completed)
        return completed.returncode
    finally:
        if temp_path:
            Path(temp_path).unlink(missing_ok=True)


def run_mutation(payload, request_path, base_dir):
    final_path = resolve_workspace_path(base_dir, require_field(payload, "file"))
    final_path.parent.mkdir(parents=True, exist_ok=True)
    temp_path = final_path.with_name(
        f".{final_path.stem}.xelora-{uuid.uuid4().hex}{final_path.suffix}"
    )
    if final_path.exists():
        shutil.copy2(final_path, temp_path)

    staged_payload = dict(payload)
    staged_payload["file"] = temp_path.relative_to(base_dir).as_posix()
    display_file = final_path.relative_to(base_dir).as_posix()
    replacements = {staged_payload["file"]: display_file}
    if str(payload.get("action")) == "write_docx":
        return run_write_docx_mutation(staged_payload, temp_path, final_path, base_dir, replacements)
    if str(payload.get("action")) == "write_xlsx":
        return run_write_xlsx_mutation(staged_payload, temp_path, final_path, base_dir, replacements)

    command, batch_path = build_officecli_command(staged_payload, str(request_path))
    try:
        completed = run_officecli(command)
        print_completed(completed, replacements)
        if completed.returncode != 0:
            return completed.returncode

        validated = run_officecli(
            ["officecli", "validate", staged_payload["file"], "--json"]
        )
        print_completed(validated, replacements)
        if validated.returncode != 0:
            print("office_validation_failed", file=sys.stderr)
            return validated.returncode or 1

        os.replace(temp_path, final_path)
        return 0
    finally:
        if batch_path:
            Path(batch_path).unlink(missing_ok=True)
        temp_path.unlink(missing_ok=True)


def normalize_paragraphs(payload):
    raw = payload.get("paragraphs")
    if raw is None:
        content = payload.get("content")
        if isinstance(content, str):
            raw = [line.strip() for line in content.splitlines() if line.strip()]
    if not isinstance(raw, list) or not raw:
        raise ValueError("paragraphs must be a non-empty array for write_docx")
    paragraphs = []
    for item in raw:
        text = str(item).strip()
        if text:
            paragraphs.append(text)
    if not paragraphs:
        raise ValueError("paragraphs must contain at least one non-empty paragraph")
    return paragraphs


def run_write_docx_mutation(payload, temp_path, final_path, base_dir, replacements):
    if final_path.suffix.lower() != ".docx":
        raise ValueError("write_docx requires a .docx file")

    title = str(payload.get("title", "")).strip()
    paragraphs = normalize_paragraphs(payload)
    staged_file = temp_path.relative_to(base_dir).as_posix()

    commands = [["officecli", "create", staged_file, "--json", "--force"]]
    if title:
        commands.append(
            [
                "officecli",
                "add",
                staged_file,
                "/",
                "--type",
                "paragraph",
                "--prop",
                f"text={title}",
                "--prop",
                "style=Title",
                "--json",
            ]
        )
    for paragraph in paragraphs:
        commands.append(
            [
                "officecli",
                "add",
                staged_file,
                "/",
                "--type",
                "paragraph",
                "--prop",
                f"text={paragraph}",
                "--json",
            ]
        )
    commands.append(["officecli", "validate", staged_file, "--json"])

    try:
        for command in commands:
            completed = run_officecli(command)
            print_completed(completed, replacements)
            if completed.returncode != 0:
                return completed.returncode
        os.replace(temp_path, final_path)
        return 0
    finally:
        temp_path.unlink(missing_ok=True)


def normalize_xlsx_sheets(payload):
    raw_sheets = payload.get("sheets")
    if raw_sheets is None:
        raw_sheets = [
            {
                "name": payload.get("sheet", "Sheet1"),
                "headers": payload.get("headers", []),
                "rows": payload.get("rows", []),
            }
        ]
    if not isinstance(raw_sheets, list) or not raw_sheets:
        raise ValueError("sheets must be a non-empty array for write_xlsx")

    sheets = []
    for index, raw_sheet in enumerate(raw_sheets, start=1):
        if not isinstance(raw_sheet, dict):
            raise ValueError("each sheet must be an object")
        name = str(raw_sheet.get("name") or f"Sheet{index}").strip()
        if not name:
            name = f"Sheet{index}"
        if len(name) > 31 or any(char in name for char in "[]:*?/\\"):
            raise ValueError(f"invalid sheet name: {name}")

        headers = raw_sheet.get("headers", [])
        rows = raw_sheet.get("rows", [])
        if not isinstance(headers, list):
            raise ValueError("headers must be an array")
        if not isinstance(rows, list):
            raise ValueError("rows must be an array")
        normalized_rows = []
        for row in rows:
            if not isinstance(row, list):
                raise ValueError("each row must be an array")
            normalized_rows.append(row)
        if not headers and not normalized_rows:
            raise ValueError("each sheet must include headers or rows")
        sheets.append({"name": name, "headers": headers, "rows": normalized_rows})
    return sheets


def xlsx_cell_value(value):
    if value is None or isinstance(value, (bool, int, float)):
        return value
    return str(value)


def run_write_xlsx_mutation(payload, temp_path, final_path, base_dir, replacements):
    if final_path.suffix.lower() != ".xlsx":
        raise ValueError("write_xlsx requires a .xlsx file")

    from openpyxl import Workbook, load_workbook
    from openpyxl.styles import Alignment, Border, Font, PatternFill, Side

    sheets = normalize_xlsx_sheets(payload)
    workbook = Workbook()
    header_fill = PatternFill(fill_type="solid", fgColor="2F5496")
    header_font = Font(bold=True, color="FFFFFF")
    thin_border = Border(bottom=Side(style="thin", color="D9E2F3"))

    for index, sheet in enumerate(sheets):
        worksheet = workbook.active if index == 0 else workbook.create_sheet()
        worksheet.title = sheet["name"]
        headers = [xlsx_cell_value(value) for value in sheet["headers"]]
        if headers:
            worksheet.append(headers)
        for row in sheet["rows"]:
            worksheet.append([xlsx_cell_value(value) for value in row])

        if headers:
            worksheet.freeze_panes = "A2"
            worksheet.auto_filter.ref = worksheet.dimensions
            for cell in worksheet[1]:
                cell.fill = header_fill
                cell.font = header_font
                cell.alignment = Alignment(horizontal="center", vertical="center", wrap_text=True)
                cell.border = thin_border
        for row in worksheet.iter_rows(min_row=2 if headers else 1):
            for cell in row:
                cell.alignment = Alignment(vertical="top", wrap_text=True)
                cell.border = thin_border

        for column_cells in worksheet.columns:
            max_length = max(len(str(cell.value)) if cell.value is not None else 0 for cell in column_cells)
            worksheet.column_dimensions[column_cells[0].column_letter].width = min(max(max_length + 2, 12), 60)

    try:
        workbook.save(temp_path)
        workbook.close()
        validated = load_workbook(temp_path, read_only=True)
        validated.close()
        os.replace(temp_path, final_path)
        display_file = final_path.relative_to(base_dir).as_posix()
        print(
            json.dumps(
                {
                    "status": "success",
                    "file": display_file,
                    "sheets": [sheet["name"] for sheet in sheets],
                },
                ensure_ascii=False,
            )
        )
        return 0
    finally:
        temp_path.unlink(missing_ok=True)


def run_request(request_arg):
    base_dir = Path.cwd().resolve()
    request_path = resolve_workspace_path(base_dir, request_arg)

    with request_path.open("r", encoding="utf-8") as handle:
        payload = json.load(handle)
    if not isinstance(payload, dict):
        raise ValueError("request payload must be a JSON object")

    action = str(require_field(payload, "action"))
    if action in MUTATING_ACTIONS:
        return run_mutation(payload, request_path, base_dir)
    return run_read_only(payload, request_path)


def main():
    if len(sys.argv) < 2:
        print("usage: officecli_bridge.py <request.json>", file=sys.stderr)
        return 2
    return run_request(sys.argv[1])


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:
        print(str(exc), file=sys.stderr)
        raise SystemExit(1)
