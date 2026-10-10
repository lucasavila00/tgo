#!/usr/bin/env python3
"""Reject numbered enum placeholders in production TGo source."""

from __future__ import annotations

from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[1]
SOURCE_ROOTS = ("cmd", "internal", "pkg")
SKIPPED_DIRECTORIES = frozenset({"fixtures", "testdata", "vendor"})
NUMBERED_ENUM_VALUE = re.compile(r"\benumValue\d+\b")


def production_sources(repository: Path) -> list[Path]:
    """Return handwritten production TGo source paths."""
    result: list[Path] = []
    for source_root in SOURCE_ROOTS:
        root = repository / source_root
        if not root.is_dir():
            continue
        for path in root.rglob("*.tgo"):
            relative = path.relative_to(repository)
            if path.name.endswith("_test.tgo"):
                continue
            if any(
                part.startswith((".", "_")) or part in SKIPPED_DIRECTORIES
                for part in relative.parts[:-1]
            ):
                continue
            result.append(relative)
    return sorted(result)


def failures(repository: Path = ROOT) -> list[str]:
    """Return numbered enum placeholders in production TGo source."""
    result: list[str] = []
    for relative in production_sources(repository):
        path = repository / relative
        for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
            if NUMBERED_ENUM_VALUE.search(line):
                result.append(
                    f"{relative}:{number}: replace numbered enum placeholder "
                    "with a role name"
                )
    return result


def main() -> int:
    """Report each numbered enum placeholder."""
    result = failures()
    for failure in result:
        print(failure)
    return 1 if result else 0


if __name__ == "__main__":
    sys.exit(main())
