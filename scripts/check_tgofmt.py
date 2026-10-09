#!/usr/bin/env python3
"""Require canonical formatting for production TGo source."""

from __future__ import annotations

import os
import subprocess
import tempfile
from pathlib import Path

from scripts import check_generated


ROOT = Path(__file__).resolve().parents[1]


def production_sources(repository: Path) -> dict[Path, bytes]:
    """Read production TGo source with the generated-source exclusions."""
    return check_generated.production_files(repository, ".tgo")


def staged_sources(repository: Path) -> dict[Path, bytes]:
    """Read staged production TGo source."""
    production = production_sources(repository)
    output = subprocess.check_output(
        [
            "git",
            "diff",
            "--cached",
            "--name-only",
            "-z",
            "--diff-filter=ACMR",
            "--",
            "*.tgo",
        ],
        cwd=repository,
    )
    paths = {
        Path(os.fsdecode(raw_path))
        for raw_path in output.split(b"\0")
        if raw_path
    }
    return {
        path: subprocess.check_output(
            ["git", "show", f":./{path.as_posix()}"],
            cwd=repository,
        )
        for path in sorted(paths & production.keys())
    }


def noncanonical_sources(sources: dict[Path, bytes]) -> list[Path]:
    """Return source paths that tgofmt changes."""
    if not sources:
        return []
    with tempfile.TemporaryDirectory(prefix="tgo-format-") as temporary:
        work = Path(temporary)
        arguments: list[str] = []
        originals: dict[str, Path] = {}
        for path, source in sorted(sources.items()):
            copy = work / path
            copy.parent.mkdir(parents=True, exist_ok=True)
            copy.write_bytes(source)
            argument = str(copy)
            arguments.append(argument)
            originals[argument] = path
        result = subprocess.run(
            ["go", "run", "./cmd/tgofmt", "-l", "--", *arguments],
            cwd=ROOT,
            check=True,
            capture_output=True,
            text=True,
        )
        return [originals[line] for line in result.stdout.splitlines()]


def failure_message(path: Path) -> str:
    """Give the exact command that fixes a source file."""
    return f"{path}: run 'go run ./cmd/tgofmt -w -- {path}'"


def main() -> None:
    sources = (
        staged_sources(ROOT)
        if os.environ.get("TGOFMT_STAGED") == "1"
        else production_sources(ROOT)
    )
    failures = noncanonical_sources(sources)
    if failures:
        commands = [failure_message(path) for path in failures]
        raise SystemExit("\n".join(commands))


if __name__ == "__main__":
    main()
