#!/usr/bin/env python3
"""Reject TGo source files that are longer than the repository limit."""

from pathlib import Path
import sys


MAX_LINES = 700
ROOT = Path(__file__).resolve().parent.parent


def line_count(path: Path) -> int:
    """Return the number of source lines in one file."""
    with path.open("rb") as source:
        return sum(1 for _ in source)


def main() -> int:
    """Report each TGo file that exceeds the line limit."""
    failures = []
    for path in sorted(ROOT.rglob("*.tgo")):
        if ".git" in path.parts:
            continue
        count = line_count(path)
        if count > MAX_LINES:
            failures.append((path.relative_to(ROOT), count))

    for path, count in failures:
        print(f"{path}: {count} lines; maximum is {MAX_LINES}")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
