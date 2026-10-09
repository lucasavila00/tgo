#!/usr/bin/env python3
"""Check that committed production Go matches current TGo output."""

from __future__ import annotations

import os
import shutil
import subprocess
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
# These names match the build driver's package walk. The copied Go corpus and
# all repository fixtures are below testdata or the explicit third-party tree.
SKIPPED_DIRECTORY_NAMES = frozenset({"bin", "testdata", "vendor", "__pycache__"})
SKIPPED_TREES = (Path("third_party/go"),)


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


def skip_directory(repository: Path, path: Path) -> bool:
    relative = path.relative_to(repository)
    name = path.name
    if name.startswith((".", "_")) or name in SKIPPED_DIRECTORY_NAMES:
        return True
    if any(relative == tree or tree in relative.parents for tree in SKIPPED_TREES):
        return True
    return relative != Path(".") and (path / "go.mod").is_file()


def production_files(repository: Path, suffix: str) -> dict[Path, bytes]:
    files: dict[Path, bytes] = {}
    for directory, names, filenames in os.walk(repository):
        path = Path(directory)
        names[:] = sorted(
            name
            for name in names
            if not skip_directory(repository, path / name)
        )
        for name in sorted(filenames):
            if not name.endswith(suffix):
                continue
            file_path = path / name
            files[file_path.relative_to(repository)] = file_path.read_bytes()
    return files


def require_equal(committed: dict[Path, bytes], generated: dict[Path, bytes]) -> None:
    failures: list[str] = []
    for path in sorted(set(committed) | set(generated)):
        if path not in committed:
            failures.append(f"missing generated output: {path}")
        elif path not in generated:
            failures.append(f"orphan generated output: {path}")
        elif committed[path] != generated[path]:
            failures.append(f"stale or edited generated output: {path}")
    if failures:
        raise SystemExit("\n".join(failures))


def main() -> None:
    with tempfile.TemporaryDirectory(prefix="tgo-bootstrap-") as temporary:
        work = Path(temporary)
        repository = copy_repository(work)
        compiler = work / "tgo"

        committed = production_files(repository, ".go")
        run(repository, "go", "build", "-o", str(compiler), "./cmd/tgo")
        run(repository, str(compiler), "build", "./...")
        generated = production_files(repository, ".go")
        require_equal(committed, generated)


if __name__ == "__main__":
    main()
