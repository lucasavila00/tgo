#!/usr/bin/env python3
"""Check that self-hosted generated Go matches its source."""

from __future__ import annotations

import shutil
import subprocess
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
GENERATED_ROOTS = (
    Path("cmd/tgofmt"),
    Path("pkg/format"),
    Path("pkg/syntax"),
    Path("internal/sourcefacts"),
    Path("internal/navigation"),
    Path("internal/tgolint"),
    Path("cmd/tgonav"),
)


def copy_repository(destination: Path) -> Path:
    repository = destination / "repository"

    def ignore(_: str, names: list[str]) -> set[str]:
        skipped = {name for name in names if name in {".git", "bin", ".tgo.lock"}}
        skipped.update(name for name in names if name == "__pycache__")
        return skipped

    shutil.copytree(ROOT, repository, ignore=ignore)
    return repository


def run(repository: Path, *command: str) -> None:
    subprocess.run(command, cwd=repository, check=True)


def generated_files(repository: Path) -> dict[Path, bytes]:
    files: dict[Path, bytes] = {}
    for relative_root in GENERATED_ROOTS:
        root = repository / relative_root
        if not root.is_dir():
            continue
        for path in sorted(root.rglob("*_tgo.go")):
            files[path.relative_to(repository)] = path.read_bytes()
    return files


def require_equal(
    expected: dict[Path, bytes],
    actual: dict[Path, bytes],
    description: str,
) -> None:
    paths = sorted(set(expected) | set(actual))
    for path in paths:
        if expected.get(path) != actual.get(path):
            raise SystemExit(f"{description}: {path}")


def main() -> None:
    with tempfile.TemporaryDirectory(prefix="tgo-bootstrap-") as temporary:
        work = Path(temporary)
        repository = copy_repository(work)
        compiler = work / "tgo"

        committed = generated_files(repository)
        run(repository, "go", "build", "-o", str(compiler), "./cmd/tgo")

        run(
            repository,
            str(compiler),
            "build",
            "./cmd/tgofmt",
            "./pkg/format",
            "./pkg/syntax",
            "./internal/sourcefacts",
            "./internal/navigation",
            "./internal/tgolint",
            "./cmd/tgonav",
        )
        generated = generated_files(repository)
        require_equal(committed, generated, "committed generated output is stale")


if __name__ == "__main__":
    main()
