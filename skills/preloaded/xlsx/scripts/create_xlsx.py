#!/usr/bin/env python3
import json
import sys
import tempfile
from pathlib import Path


def load_officecli_bridge():
    preloaded_dir = Path(__file__).resolve().parents[2]
    bridge_dir = preloaded_dir / "officecli-document-editing" / "scripts"
    sys.path.insert(0, str(bridge_dir))
    import officecli_bridge

    return officecli_bridge


def prepare_request(request_path):
    path = Path(request_path)
    with path.open("r", encoding="utf-8-sig") as handle:
        payload = json.load(handle)
    if not isinstance(payload, dict):
        raise ValueError("request payload must be a JSON object")

    if payload.get("action") == "write_xlsx":
        return path, None

    normalized = dict(payload)
    normalized["action"] = "write_xlsx"
    if not normalized.get("file"):
        normalized["file"] = "output.xlsx"

    temp = tempfile.NamedTemporaryFile(
        "w",
        encoding="utf-8",
        suffix=".json",
        prefix="create-xlsx-",
        dir=path.parent,
        delete=False,
    )
    try:
        json.dump(normalized, temp, ensure_ascii=False)
        temp.write("\n")
        temp.close()
        return Path(temp.name), Path(temp.name)
    except Exception:
        temp.close()
        Path(temp.name).unlink(missing_ok=True)
        raise


def run(request_arg):
    officecli_bridge = load_officecli_bridge()
    request_path, cleanup_path = prepare_request(request_arg)
    try:
        bridge_arg = request_path.resolve().relative_to(Path.cwd().resolve()).as_posix()
        return officecli_bridge.run_request(bridge_arg)
    finally:
        if cleanup_path:
            cleanup_path.unlink(missing_ok=True)


def main():
    if len(sys.argv) < 2:
        print("usage: create_xlsx.py <request.json>", file=sys.stderr)
        return 2
    return run(sys.argv[1])


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:
        print(str(exc), file=sys.stderr)
        raise SystemExit(1)
