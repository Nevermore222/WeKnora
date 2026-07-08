#!/usr/bin/env python3
import json
import os
import shlex
import sys
import time
from pathlib import Path

from e2b.sandbox.filesystem.filesystem import FileType
from e2b.sandbox_sync.commands.command_handle import CommandExitException
from e2b_code_interpreter import Sandbox


IGNORE_DIRS = {".git", "node_modules", "__pycache__"}
IGNORE_FILE_SUFFIXES = {".pyc", ".pyo", ".lock", ".log"}
REMOTE_ROOT = "/workspace"


def should_ignore_local(path: Path, root: Path) -> bool:
    rel = path.relative_to(root)
    for part in rel.parts[:-1]:
        if part in IGNORE_DIRS:
            return True
    name = rel.name
    if name.startswith("."):
        return True
    return any(name.endswith(suffix) for suffix in IGNORE_FILE_SUFFIXES)


def upload_workspace(sandbox: Sandbox, local_root: Path) -> None:
    sandbox.files.make_dir(REMOTE_ROOT)
    for path in sorted(local_root.rglob("*")):
        if should_ignore_local(path, local_root):
            continue
        remote_path = f"{REMOTE_ROOT}/{path.relative_to(local_root).as_posix()}"
        if path.is_dir():
            sandbox.files.make_dir(remote_path)
            continue
        if path.is_file():
            sandbox.files.write(remote_path, path.read_bytes(), use_octet_stream=True)


def sync_workspace_back(sandbox: Sandbox, local_root: Path) -> None:
    for entry in walk_remote_files(sandbox, REMOTE_ROOT):
        rel = entry.path.removeprefix(f"{REMOTE_ROOT}/")
        if not rel:
            continue
        if rel.startswith("."):
            continue
        local_path = local_root / Path(rel)
        local_path.parent.mkdir(parents=True, exist_ok=True)
        local_path.write_bytes(sandbox.files.read(entry.path, format="bytes"))


def walk_remote_files(sandbox: Sandbox, remote_dir: str):
    for entry in sandbox.files.list(remote_dir):
        if entry.type == FileType.DIR:
            yield from walk_remote_files(sandbox, entry.path)
        elif entry.type == FileType.FILE:
            yield entry


def detect_interpreter(script_path: str) -> str:
    suffix = Path(script_path).suffix.lower()
    if suffix == ".py":
        return "python3"
    if suffix in {".sh", ".bash"}:
        return "bash"
    if suffix == ".js":
        return "node"
    if suffix == ".ts":
        return "tsx"
    return "sh"


def build_command(request: dict) -> str:
    script_path = request["script_path"]
    local_root = Path(request["base_path"]).resolve()
    remote_script = f"{REMOTE_ROOT}/{Path(script_path).resolve().relative_to(local_root).as_posix()}"
    command_parts = [detect_interpreter(script_path), remote_script]
    command_parts.extend(request.get("args") or [])
    command = " ".join(shlex.quote(part) for part in command_parts)

    stdin_text = request.get("stdin") or ""
    if stdin_text:
        stdin_remote = f"{REMOTE_ROOT}/.xelora-stdin.txt"
        request["_stdin_remote"] = stdin_remote
        command += " < " + shlex.quote(stdin_remote)
    return command


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: cubesandbox_exec.py <request.json>", file=sys.stderr)
        return 2

    request_path = Path(sys.argv[1]).resolve()
    request = json.loads(request_path.read_text(encoding="utf-8"))
    local_root = Path(request["base_path"]).resolve()

    start = time.monotonic()
    with Sandbox.create(template=os.environ["CUBE_TEMPLATE_ID"]) as sandbox:
        upload_workspace(sandbox, local_root)

        stdin_text = request.get("stdin") or ""
        if stdin_text:
            sandbox.files.write(f"{REMOTE_ROOT}/.xelora-stdin.txt", stdin_text)

        command = build_command(request)
        response = {
            "stdout": "",
            "stderr": "",
            "exit_code": 0,
            "error": "",
            "duration_ms": 0,
            "sandbox_id": getattr(sandbox, "sandbox_id", ""),
        }

        try:
            result = sandbox.commands.run(
                command,
                cwd=REMOTE_ROOT,
                timeout=request.get("timeout_sec") or 60,
            )
            response["stdout"] = result.stdout
            response["stderr"] = result.stderr
            response["exit_code"] = result.exit_code
            response["error"] = result.error or ""
        except CommandExitException as exc:
            response["stdout"] = exc.stdout
            response["stderr"] = exc.stderr
            response["exit_code"] = exc.exit_code
            response["error"] = exc.error or str(exc)

        if stdin_text:
            sandbox.files.remove(f"{REMOTE_ROOT}/.xelora-stdin.txt")
        sync_workspace_back(sandbox, local_root)
        response["duration_ms"] = int((time.monotonic() - start) * 1000)
        print(json.dumps(response))
        return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:
        print(str(exc), file=sys.stderr)
        raise
