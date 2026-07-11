#!/usr/bin/env python3
"""OpenSandbox helper for executor-backed skill execution.

This helper keeps the integration boundary thin: Xelora owns job, workspace,
and artifact semantics, while OpenSandbox provides remote sandbox lifecycle and
command execution. The helper uploads the prepared local workspace, runs the
requested script under /workspace, then downloads the resulting workspace back
to the local skill directory.
"""

from __future__ import annotations

import json
import base64
import io
import os
import shlex
import sys
import time
import tarfile
import traceback
from datetime import timedelta
from pathlib import Path
from urllib.parse import urlparse

import httpx
from opensandbox import SandboxSync
from opensandbox.config import ConnectionConfigSync
from opensandbox.models.filesystem import WriteEntry


WORKSPACE_ROOT = "/workspace"
UPLOAD_ARCHIVE_PATH = "/tmp/xelora-workspace.tar.gz.b64"
DOWNLOAD_SENTINEL = "__XELORA_WORKSPACE_ARCHIVE__="
EXIT_SENTINEL = "__XELORA_EXIT_CODE__="
STDIN_PATH = f"{WORKSPACE_ROOT}/.xelora-stdin.txt"
IGNORE_DIRS = {".git", "node_modules", "__pycache__"}
IGNORE_FILE_SUFFIXES = {".pyc", ".pyo", ".lock", ".log"}


def _load_request(request_path: Path) -> dict:
    return json.loads(request_path.read_text(encoding="utf-8"))


def _parse_connection(base_url: str) -> tuple[str, str]:
    raw = base_url.strip()
    if "://" not in raw:
        raw = f"http://{raw}"
    parsed = urlparse(raw)
    protocol = parsed.scheme or "http"
    domain = parsed.netloc or parsed.path
    if not domain:
        raise ValueError("invalid OpenSandbox base URL")
    return protocol, domain


def _build_connection_config() -> ConnectionConfigSync:
    base_url = os.environ.get("XELORA_OPENSANDBOX_BASE_URL", "").strip()
    api_key = os.environ.get("XELORA_OPENSANDBOX_API_KEY", "").strip()
    if not base_url:
        raise ValueError("XELORA_OPENSANDBOX_BASE_URL is required")
    if not api_key:
        raise ValueError("XELORA_OPENSANDBOX_API_KEY is required")
    protocol, domain = _parse_connection(base_url)
    return ConnectionConfigSync(
        domain=domain,
        protocol=protocol,
        api_key=api_key,
        request_timeout=timedelta(seconds=180),
        use_server_proxy=True,
        transport=httpx.HTTPTransport(limits=httpx.Limits(max_connections=20)),
    )


def _should_ignore_local(path: Path, root: Path) -> bool:
    rel = path.relative_to(root)
    for part in rel.parts[:-1]:
        if part in IGNORE_DIRS:
            return True
    name = rel.name
    if name.startswith("."):
        return True
    return any(name.endswith(suffix) for suffix in IGNORE_FILE_SUFFIXES)


def _create_workspace_archive(base_path: Path) -> str:
    buffer = io.BytesIO()
    with tarfile.open(fileobj=buffer, mode="w:gz") as archive:
        for path in sorted(base_path.rglob("*")):
            if _should_ignore_local(path, base_path):
                continue
            archive.add(path, arcname=path.relative_to(base_path).as_posix(), recursive=False)
    return base64.b64encode(buffer.getvalue()).decode("ascii")


def _restore_workspace_archive(encoded: str, base_path: Path) -> None:
    data = base64.b64decode(encoded.encode("ascii"))
    base_path.mkdir(parents=True, exist_ok=True)
    with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as archive:
        archive.extractall(base_path)


def _detect_runtime(script_path: str) -> list[str]:
    suffix = Path(script_path).suffix.lower()
    if suffix == ".py":
        return ["python3", script_path]
    if suffix == ".js":
        return ["node", script_path]
    if suffix == ".ts":
        return ["tsx", script_path]
    if suffix == ".sh":
        return ["bash", script_path]
    return [script_path]


def _entrypoint_for_image(image: str) -> list[str] | None:
    normalized = image.lower()
    if "code-interpreter" in normalized:
        return ["/opt/code-interpreter/code-interpreter.sh"]
    return None


def _join_logs(items: object) -> str:
    if not items:
        return ""
    chunks: list[str] = []
    for item in items:
        text = getattr(item, "text", "")
        if text:
            chunks.append(text)
    return "".join(chunks)


def _write_workspace_archive(sandbox: SandboxSync, encoded_archive: str) -> None:
    sandbox.files.write_files(
        [
            WriteEntry(
                path=UPLOAD_ARCHIVE_PATH,
                data=encoded_archive,
                mode=644,
            )
        ]
    )


def _extract_workspace_archive(sandbox: SandboxSync) -> None:
    command = (
        "python3 - <<'PY'\n"
        "import base64, pathlib, tarfile\n"
        "src = pathlib.Path('/tmp/xelora-workspace.tar.gz.b64')\n"
        "workspace = pathlib.Path('/workspace')\n"
        "workspace.mkdir(parents=True, exist_ok=True)\n"
        "data = base64.b64decode(src.read_text())\n"
        "archive = pathlib.Path('/tmp/xelora-workspace.tar.gz')\n"
        "archive.write_bytes(data)\n"
        "with tarfile.open(archive, 'r:gz') as tf:\n"
        "    tf.extractall(workspace)\n"
        "PY"
    )
    sandbox.commands.run(command)


def _run_skill_command(sandbox: SandboxSync, request: dict) -> tuple[str, str, int]:
    relative_script = str(Path(request["script_path"]).resolve().relative_to(Path(request["base_path"]).resolve()))
    command_parts = _detect_runtime(relative_script) + list(request.get("args", []))
    quoted = " ".join(shlex.quote(part) for part in command_parts)
    stdin_redirect = ""
    if request.get("stdin"):
        stdin_redirect = f" < {shlex.quote(STDIN_PATH)}"
    wrapped = (
        f"sh -lc 'cd {WORKSPACE_ROOT} && {quoted}{stdin_redirect}; "
        f"code=$?; printf \"\\n{EXIT_SENTINEL}%s\\n\" \"$code\"; exit 0'"
    )
    execution = sandbox.commands.run(wrapped)
    stdout = _join_logs(getattr(execution.logs, "stdout", []))
    stderr = _join_logs(getattr(execution.logs, "stderr", []))

    exit_code = 1
    marker_index = stdout.rfind(EXIT_SENTINEL)
    if marker_index >= 0:
        marker = stdout[marker_index + len(EXIT_SENTINEL) :].splitlines()[0].strip()
        try:
            exit_code = int(marker)
        except ValueError:
            exit_code = 1
        stdout = stdout[:marker_index].rstrip("\n")
    return stdout, stderr, exit_code


def _download_workspace_archive(sandbox: SandboxSync) -> str:
    command = (
        "python3 - <<'PY'\n"
        "import base64, io, pathlib, tarfile\n"
        "buf = io.BytesIO()\n"
        "with tarfile.open(fileobj=buf, mode='w:gz') as tf:\n"
        "    tf.add('/workspace', arcname='.', recursive=True)\n"
        f"print('{DOWNLOAD_SENTINEL}' + base64.b64encode(buf.getvalue()).decode('ascii'))\n"
        "PY"
    )
    execution = sandbox.commands.run(command)
    stdout = _join_logs(getattr(execution.logs, "stdout", []))
    marker_index = stdout.find(DOWNLOAD_SENTINEL)
    if marker_index < 0:
        raise RuntimeError("failed to capture workspace archive from OpenSandbox")
    return stdout[marker_index + len(DOWNLOAD_SENTINEL) :].strip()


def _write_stdin_file(sandbox: SandboxSync, stdin_text: str) -> None:
    if not stdin_text:
        return
    sandbox.files.write_files(
        [
            WriteEntry(
                path=STDIN_PATH,
                data=stdin_text,
                mode=644,
            )
        ]
    )


def _wait_until_ready(sandbox: SandboxSync, timeout_sec: int) -> None:
    deadline = time.time() + timeout_sec
    last_error: Exception | None = None

    while time.time() < deadline:
        try:
            sandbox.commands.run("true")
            return
        except Exception as exc:  # noqa: BLE001
            last_error = exc
            time.sleep(2)

    if last_error is not None:
        raise RuntimeError(f"sandbox command channel did not become ready: {last_error}") from last_error
    raise RuntimeError("sandbox command channel did not become ready before timeout")


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: opensandbox_exec.py <request.json>", file=sys.stderr)
        return 2

    request_path = Path(sys.argv[1])
    payload = _load_request(request_path)
    start = time.time()
    sandbox = None

    try:
        config = _build_connection_config()
        image = os.environ.get("XELORA_OPENSANDBOX_TEMPLATE_ID", "").strip()
        if not image:
            raise ValueError("XELORA_OPENSANDBOX_TEMPLATE_ID is required")

        base_path = Path(payload["base_path"]).resolve()
        encoded_archive = _create_workspace_archive(base_path)
        entrypoint = _entrypoint_for_image(image)
        sandbox = SandboxSync.create(
            image,
            entrypoint=entrypoint,
            connection_config=config,
            timeout=timedelta(seconds=int(payload.get("timeout_sec", 60))),
            skip_health_check=True,
        )
        with sandbox:
            _wait_until_ready(sandbox, int(payload.get("timeout_sec", 60)))
            _write_workspace_archive(sandbox, encoded_archive)
            _extract_workspace_archive(sandbox)
            _write_stdin_file(sandbox, payload.get("stdin", ""))
            stdout, stderr, exit_code = _run_skill_command(sandbox, payload)
            downloaded_archive = _download_workspace_archive(sandbox)
            _restore_workspace_archive(downloaded_archive, base_path)

        response = {
            "stdout": stdout,
            "stderr": stderr,
            "exit_code": exit_code,
            "error": "" if exit_code == 0 else "script execution failed",
            "duration_ms": int((time.time() - start) * 1000),
            "sandbox_id": getattr(sandbox, "id", ""),
        }
    except Exception as exc:  # noqa: BLE001
        response = {
            "stdout": "",
            "stderr": traceback.format_exc(),
            "exit_code": 1,
            "error": str(exc),
            "duration_ms": int((time.time() - start) * 1000),
            "sandbox_id": getattr(sandbox, "id", "") if sandbox is not None else "",
        }
    finally:
        if sandbox is not None:
            try:
                sandbox.kill()
            except Exception:  # noqa: BLE001
                pass

    json.dump(response, sys.stdout)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
