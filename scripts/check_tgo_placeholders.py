#!/usr/bin/env python3
"""Reject numbered enum placeholders in production TGo source."""

from __future__ import annotations

from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[1]
SOURCE_ROOTS = ("cmd", "internal", "pkg")
SKIPPED_DIRECTORIES = frozenset({"fixtures", "testdata", "vendor"})
IDENTIFIER = re.compile(r"[^\W\d]\w*|_\w*")
NUMBERED_ENUM_VALUE = re.compile(r"enumValue\d+")
NON_CODE = re.compile(
    r'''//[^\n]*|/\*.*?\*/|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|`[^`]*`''',
    re.DOTALL,
)


def code_text(source: str) -> str:
    """Replace comments and literals with spaces, but keep line breaks."""
    return NON_CODE.sub(
        lambda match: re.sub(r"[^\n]", " ", match.group()),
        source,
    )


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
        source = code_text(path.read_text(encoding="utf-8"))
        for number, line in enumerate(source.splitlines(), 1):
            if any(
                NUMBERED_ENUM_VALUE.fullmatch(match.group())
                for match in IDENTIFIER.finditer(line)
            ):
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
