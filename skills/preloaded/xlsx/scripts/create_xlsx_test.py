import json
import os
import tempfile
import unittest
from pathlib import Path

import openpyxl

import create_xlsx


class CreateXlsxCompatTest(unittest.TestCase):
    def test_request_without_action_creates_workbook(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            request = root / "request.json"
            request.write_text(
                json.dumps(
                    {
                        "file": "summary.xlsx",
                        "sheets": [
                            {
                                "name": "\u9a8c\u8bc1",
                                "headers": ["\u9879\u76ee", "\u72b6\u6001"],
                                "rows": [["create_xlsx", "\u517c\u5bb9\u6210\u529f"]],
                            }
                        ],
                    },
                    ensure_ascii=False,
                ),
                encoding="utf-8",
            )

            previous_cwd = Path.cwd()
            try:
                os.chdir(root)
                result = create_xlsx.run("request.json")
            finally:
                os.chdir(previous_cwd)

            self.assertEqual(result, 0)
            workbook = openpyxl.load_workbook(root / "summary.xlsx")
            try:
                worksheet = workbook["\u9a8c\u8bc1"]
                self.assertEqual(worksheet["A1"].value, "\u9879\u76ee")
                self.assertEqual(worksheet["B1"].value, "\u72b6\u6001")
                self.assertEqual(worksheet["A2"].value, "create_xlsx")
                self.assertEqual(worksheet["B2"].value, "\u517c\u5bb9\u6210\u529f")
            finally:
                workbook.close()

    def test_write_xlsx_action_request_can_use_relative_markdown_path(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            request = root / "generated-input.md"
            request.write_text(
                json.dumps(
                    {
                        "action": "write_xlsx",
                        "file": "from-generated-input.xlsx",
                        "sheets": [
                            {
                                "name": "\u9a8c\u8bc1",
                                "headers": ["\u9879\u76ee", "\u72b6\u6001"],
                                "rows": [["generated-md", "\u76f4\u901a\u6210\u529f"]],
                            }
                        ],
                    },
                    ensure_ascii=False,
                ),
                encoding="utf-8",
            )

            previous_cwd = Path.cwd()
            try:
                os.chdir(root)
                result = create_xlsx.run("generated-input.md")
            finally:
                os.chdir(previous_cwd)

            self.assertEqual(result, 0)
            workbook = openpyxl.load_workbook(root / "from-generated-input.xlsx")
            try:
                worksheet = workbook["\u9a8c\u8bc1"]
                self.assertEqual(worksheet["A2"].value, "generated-md")
                self.assertEqual(worksheet["B2"].value, "\u76f4\u901a\u6210\u529f")
            finally:
                workbook.close()


if __name__ == "__main__":
    unittest.main()
