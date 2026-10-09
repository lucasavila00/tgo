#!/usr/bin/env python3
"""Reject oversized handwritten Go and TGo source files."""

from __future__ import annotations

from functools import cache
from pathlib import Path
import subprocess
import sys


MAX_LINES = 700
ROOT = Path(__file__).resolve().parent.parent
FIXTURE_EXCLUSIONS = {
    Path("tests/tgolint/testdata/bad/bad.go"): (
        "The fixture keeps all diagnostics and its golden output in one source file."
    ),
}


def line_count(path: Path) -> int:
    """Return the number of source lines in one file."""
    with path.open("rb") as source:
        return sum(1 for _ in source)


@cache
def target_words() -> tuple[frozenset[str], frozenset[str]]:
    """Return target words from the active Go toolchain."""
    result = subprocess.run(
        ["go", "tool", "dist", "list"],
        check=True,
        capture_output=True,
        text=True,
    )
    pairs = [line.split("/", 1) for line in result.stdout.splitlines()]
    return (
        frozenset(pair[0] for pair in pairs),
        frozenset(pair[1] for pair in pairs),
    )


def is_generated_go(path: Path) -> bool:
    """Report whether a path is in the reserved TGo output namespace."""
    if path.suffix != ".go":
        return False
    stem = path.stem
    if stem.endswith("_test"):
        stem = stem.removesuffix("_test")
    if stem.endswith("_tgo"):
        return True
    marker = stem.rfind("_tgo_")
    if marker < 0:
        return False
    target = stem[marker + len("_tgo_"):].split("_")
    target_oses, target_architectures = target_words()
    if len(target) == 1:
        return target[0] in target_oses or target[0] in target_architectures
    if len(target) == 2:
        return target[0] in target_oses and target[1] in target_architectures
    return False


def is_source(path: Path) -> bool:
    """Report whether a path is handwritten Go or TGo source."""
    return path.suffix == ".tgo" or (
        path.suffix == ".go" and not is_generated_go(path)
    )


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
