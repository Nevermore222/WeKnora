#!/usr/bin/env python3

from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

import workspace_file_writer as writer


class WorkspaceFileWriterTests(unittest.TestCase):
    def test_resolve_relative_path_rejects_escape(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            base_dir = Path(temp_dir)
            with self.assertRaises(SystemExit):
                writer.resolve_relative_path(base_dir, "../escape.md")

    def test_handle_write_and_append_create_real_markdown(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            base_dir = Path(temp_dir)
            payload = {
                "file": "reports/demo.md",
                "content": "# Hello\n",
                "overwrite": True,
            }
            writer.handle_write(base_dir, payload)
            writer.handle_append(base_dir, {"file": "reports/demo.md", "content": "World\n"})
            self.assertEqual(
                (base_dir / "reports" / "demo.md").read_text(encoding="utf-8"),
                "# Hello\nWorld\n",
            )

    def test_handle_write_json_formats_output(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            base_dir = Path(temp_dir)
            payload = {
                "file": "exports/summary.json",
                "data": {"project": "Xelora", "status": "active"},
                "indent": 2,
            }
            writer.handle_write_json(base_dir, payload)
            written = json.loads((base_dir / "exports" / "summary.json").read_text(encoding="utf-8"))
            self.assertEqual(written["project"], "Xelora")
            self.assertEqual(written["status"], "active")

    def test_load_request_accepts_inline_json_argument(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            base_dir = Path(temp_dir)
            payload = writer.load_request(
                base_dir,
                json.dumps(
                    {
                        "action": "write",
                        "file": "reports/demo.md",
                        "content": "hello\n",
                        "overwrite": True,
                    },
                    ensure_ascii=False,
                ),
            )
            self.assertEqual(payload["action"], "write")
            self.assertEqual(payload["file"], "reports/demo.md")

    def test_main_uses_default_request_json_when_no_arg_is_provided(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            base_dir = Path(temp_dir)
            (base_dir / "request.json").write_text(
                json.dumps(
                    {
                        "action": "write",
                        "file": "reports/default.md",
                        "content": "from default\n",
                        "overwrite": True,
                    },
                    ensure_ascii=False,
                ),
                encoding="utf-8",
            )
            previous_cwd = Path.cwd()
            try:
                import os

                os.chdir(base_dir)
                exit_code = writer.main(["workspace_file_writer.py"])
            finally:
                os.chdir(previous_cwd)

            self.assertEqual(exit_code, 0)
            self.assertEqual(
                (base_dir / "reports" / "default.md").read_text(encoding="utf-8"),
                "from default\n",
            )

    def test_main_reads_staged_request_from_hidden_job_directory(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            base_dir = Path(temp_dir)
            request_path = base_dir / ".xelora" / "jobs" / "job-1" / "request.json"
            request_path.parent.mkdir(parents=True)
            request_path.write_text(
                json.dumps({"action": "write", "file": "report.md", "content": "workspace\n"}),
                encoding="utf-8",
            )
            previous_cwd = Path.cwd()
            try:
                import os

                os.chdir(base_dir)
                exit_code = writer.main(
                    ["workspace_file_writer.py", ".xelora/jobs/job-1/request.json"]
                )
            finally:
                os.chdir(previous_cwd)

            self.assertEqual(exit_code, 0)
            self.assertEqual((base_dir / "report.md").read_text(encoding="utf-8"), "workspace\n")


if __name__ == "__main__":
    unittest.main()
