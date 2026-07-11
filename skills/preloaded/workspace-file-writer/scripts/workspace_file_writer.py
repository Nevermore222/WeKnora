#!/usr/bin/env python3
"""Structured file writer for markdown, text, json, and csv artifacts."""

from __future__ import annotations

import json
import shutil
import sys
from pathlib import Path


def fail(message: str, exit_code: int = 1) -> None:
    print(message, file=sys.stderr)
    raise SystemExit(exit_code)


def resolve_relative_path(base_dir: Path, candidate: str) -> Path:
    raw = (candidate or "").strip()
    if not raw:
        fail("file path is required")

    path = Path(raw)
    if path.is_absolute():
        fail(f"absolute paths are not allowed: {candidate}")

    resolved = (base_dir / path).resolve()
    try:
        resolved.relative_to(base_dir.resolve())
    except ValueError:
        fail(f"path escapes skill workspace: {candidate}")

    return resolved


def ensure_parent(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def display_relative_path(base_dir: Path, target: Path) -> str:
    return str(target.resolve().relative_to(base_dir.resolve()))


def load_request(base_dir: Path, request_arg: str) -> dict:
    trimmed = (request_arg or "").strip()
    if trimmed.startswith("{"):
        try:
            payload = json.loads(trimmed)
        except json.JSONDecodeError as exc:
            fail(f"request JSON is invalid: {exc}")
        if not isinstance(payload, dict):
            fail("request JSON must be an object")
        return payload

    request_path = resolve_relative_path(base_dir, request_arg)
    if not request_path.exists():
        fail(f"request file does not exist: {request_arg}")

    try:
        payload = json.loads(request_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        fail(f"request JSON is invalid: {exc}")

    if not isinstance(payload, dict):
        fail("request JSON must be an object")
    return payload


def handle_write(base_dir: Path, payload: dict) -> int:
    target = resolve_relative_path(base_dir, str(payload.get("file", "")))
    overwrite = bool(payload.get("overwrite", True))
    if target.exists() and not overwrite:
        fail(f"target file already exists: {target.name}")

    content = payload.get("content", "")
    if not isinstance(content, str):
        fail("content must be a string")

    ensure_parent(target)
    target.write_text(content, encoding="utf-8", newline="")
    print(f"Wrote file: {display_relative_path(base_dir, target)}")
    return 0


def handle_append(base_dir: Path, payload: dict) -> int:
    target = resolve_relative_path(base_dir, str(payload.get("file", "")))
    content = payload.get("content", "")
    if not isinstance(content, str):
        fail("content must be a string")

    ensure_parent(target)
    with target.open("a", encoding="utf-8", newline="") as handle:
        handle.write(content)
    print(f"Appended file: {display_relative_path(base_dir, target)}")
    return 0


def handle_write_json(base_dir: Path, payload: dict) -> int:
    target = resolve_relative_path(base_dir, str(payload.get("file", "")))
    indent = payload.get("indent", 2)
    if not isinstance(indent, int) or indent < 0:
        fail("indent must be a non-negative integer")

    if "data" not in payload:
        fail("data is required for write_json")

    ensure_parent(target)
    target.write_text(
        json.dumps(payload["data"], ensure_ascii=False, indent=indent) + "\n",
        encoding="utf-8",
        newline="",
    )
    print(f"Wrote JSON file: {display_relative_path(base_dir, target)}")
    return 0


def handle_copy(base_dir: Path, payload: dict) -> int:
    source = resolve_relative_path(base_dir, str(payload.get("source_file", "")))
    target = resolve_relative_path(base_dir, str(payload.get("file", "")))
    overwrite = bool(payload.get("overwrite", True))

    if not source.exists():
        fail(f"source file does not exist: {payload.get('source_file', '')}")
    if target.exists() and not overwrite:
        fail(f"target file already exists: {target.name}")

    ensure_parent(target)
    shutil.copyfile(source, target)
    print(
        "Copied file: "
        f"{display_relative_path(base_dir, source)} -> {display_relative_path(base_dir, target)}"
    )
    return 0


def main(argv: list[str]) -> int:
    base_dir = Path.cwd().resolve()
    if len(argv) == 2:
        payload = load_request(base_dir, argv[1])
    elif len(argv) == 1:
        default_request = base_dir / "request.json"
        if not default_request.exists():
            fail("usage: workspace_file_writer.py <request.json>")
        payload = load_request(base_dir, "request.json")
    else:
        fail("usage: workspace_file_writer.py <request.json>")

    action = str(payload.get("action", "")).strip().lower()

    handlers = {
        "write": handle_write,
        "append": handle_append,
        "write_json": handle_write_json,
        "copy": handle_copy,
    }
    handler = handlers.get(action)
    if handler is None:
        fail(f"unsupported action: {action}")
    return handler(base_dir, payload)


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
