import json
import os
import tempfile
import unittest
from pathlib import Path

import openpyxl

import create_xlsx


class RootCreateXlsxCompatTest(unittest.TestCase):
    def test_root_wrapper_delegates_to_scripts_create_xlsx(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            request = root / "request.json"
            request.write_text(
                json.dumps(
                    {
                        "file": "root-wrapper.xlsx",
                        "sheets": [
                            {
                                "name": "\u9a8c\u8bc1",
                                "headers": ["\u9879\u76ee"],
                                "rows": [["root-wrapper"]],
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
            workbook = openpyxl.load_workbook(root / "root-wrapper.xlsx")
            try:
                self.assertEqual(workbook["\u9a8c\u8bc1"]["A2"].value, "root-wrapper")
            finally:
                workbook.close()


if __name__ == "__main__":
    unittest.main()
