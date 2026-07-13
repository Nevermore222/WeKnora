#!/usr/bin/env python3
import importlib.util
import sys
from pathlib import Path


script_path = Path(__file__).resolve().parent / "scripts" / "create_xlsx.py"
spec = importlib.util.spec_from_file_location("xlsx_scripts_create_xlsx", script_path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

main = module.main
run = module.run


if __name__ == "__main__":
    raise SystemExit(main())
