#!/usr/bin/env python3
"""Reject numbered enum placeholders in production TGo source."""

from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parent.parent
SOURCE_ROOTS = ("cmd", "internal", "pkg")
NUMBERED_ENUM_VALUE = re.compile(r"\benumValue\d+\b")


def main() -> int:
    """Report each numbered enum placeholder in production TGo source."""
    failures: list[str] = []
    for source_root in SOURCE_ROOTS:
        for path in sorted((ROOT / source_root).rglob("*.tgo")):
            if "testdata" in path.parts or path.name.endswith("_test.tgo"):
                continue
            for number, line in enumerate(path.read_text().splitlines(), 1):
                if NUMBERED_ENUM_VALUE.search(line):
                    failures.append(f"{path.relative_to(ROOT)}:{number}")

    for failure in failures:
        print(f"{failure}: replace numbered enum placeholder with a role name")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
