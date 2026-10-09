#!/usr/bin/env python3
"""Reject oversized handwritten Go and TGo source files."""

from __future__ import annotations

from pathlib import Path
import sys


MAX_LINES = 700
ROOT = Path(__file__).resolve().parent.parent
FIXTURE_EXCLUSIONS = {
    Path("tests/tgolint/testdata/bad/bad.go"):
        "The fixture keeps all bad tgolint cases in one source file.",
}


def line_count(path: Path) -> int:
    """Return the number of source lines in one file."""
    with path.open("rb") as source:
        return sum(1 for _ in source)


def is_source(path: Path) -> bool:
    """Report whether a path is handwritten Go or TGo source."""
    if path.suffix == ".tgo":
        return True
    return path.suffix == ".go" and not path.name.endswith("_tgo.go")


def is_fixture(path: Path) -> bool:
    """Report whether a path is in a test fixture tree."""
    return "testdata" in path.parts or "fixtures" in path.parts


def failures(
    repository: Path,
    exclusions: dict[Path, str] = FIXTURE_EXCLUSIONS,
) -> list[str]:
    """Return source-size and exclusion failures for one repository."""
    result: list[str] = []
    active_exclusions: set[Path] = set()
    for relative, reason in sorted(exclusions.items()):
        path = repository / relative
        if not reason.strip():
            result.append(f"{relative}: fixture exclusion needs a reason")
            continue
        if not is_fixture(relative) or not is_source(relative):
            result.append(f"{relative}: fixture exclusion is not a source fixture")
            continue
        if not path.is_file():
            result.append(f"{relative}: stale fixture exclusion; file does not exist")
            continue
        count = line_count(path)
        if count <= MAX_LINES:
            result.append(
                f"{relative}: stale fixture exclusion; {count} lines do not exceed "
                f"{MAX_LINES}"
            )
            continue
        active_exclusions.add(relative)

    for path in sorted(repository.rglob("*")):
        if not path.is_file() or not is_source(path) or ".git" in path.parts:
            continue
        relative = path.relative_to(repository)
        if relative in active_exclusions:
            continue
        count = line_count(path)
        if count > MAX_LINES:
            result.append(f"{relative}: {count} lines; maximum is {MAX_LINES}")
    return result


def main() -> int:
    """Report each source-size and exclusion failure."""
    result = failures(ROOT)
    for failure in result:
        print(failure)
    return 1 if result else 0


if __name__ == "__main__":
    sys.exit(main())
