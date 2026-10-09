#!/usr/bin/env python3
"""Classify formatter changes and enforce the formatter corpus result."""

from __future__ import annotations

import os
import subprocess
import sys
from collections.abc import Iterable
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DEPENDENCY_FILES = frozenset(
    {
        ".github/workflows/formatter-ci.yml",
        ".github/workflows/slow-ci.yml",
        "Makefile",
        "go.mod",
        "go.sum",
        "scripts/check_formatter_boundary.py",
        "scripts/formatter_ci.py",
        "scripts/formatter_ci_test.py",
    }
)
DEPENDENCY_DIRECTORIES = (
    "cmd/tgofmt/",
    "internal/outputname/",
    "internal/packagelanguage/",
    "pkg/format/",
    "pkg/syntax/",
)


def affects_formatter(paths: Iterable[str]) -> bool:
    """Report whether changed paths can affect formatter corpus results."""
    return any(
        path in DEPENDENCY_FILES
        or any(path.startswith(directory) for directory in DEPENDENCY_DIRECTORIES)
        for path in paths
    )


def changed_paths(repository: Path, base: str, head: str) -> list[str]:
    """Return paths changed between two commits without rename folding."""
    output = subprocess.check_output(
        [
            "git",
            "diff",
            "--name-only",
            "-z",
            "--no-renames",
            f"{base}...{head}",
            "--",
        ],
        cwd=repository,
    )
    return [os.fsdecode(path) for path in output.split(b"\0") if path]


def gate_failure(
    classifier_result: str,
    formatter_required: str,
    corpus_result: str,
) -> str | None:
    """Return an error when job results do not prove the required work."""
    if classifier_result != "success":
        return f"formatter change classification ended with {classifier_result}"
    if formatter_required == "true":
        if corpus_result != "success":
            return f"required formatter corpus ended with {corpus_result}"
        return None
    if formatter_required == "false":
        if corpus_result != "skipped":
            return f"unneeded formatter corpus ended with {corpus_result}"
        return None
    return f"formatter change classification returned {formatter_required!r}"


def local_formatter_packages(repository: Path) -> list[Path]:
    """Return local package directories in the formatter dependency closure."""
    output = subprocess.check_output(
        [
            "go",
            "list",
            "-deps",
            "-f",
            "{{.Dir}}",
            "./pkg/format",
            "./cmd/tgofmt",
        ],
        cwd=repository,
        text=True,
    )
    packages: list[Path] = []
    for value in output.splitlines():
        try:
            relative = Path(value).resolve().relative_to(repository.resolve())
        except ValueError:
            continue
        packages.append(relative)
    return packages


def repository_failures(repository: Path) -> list[str]:
    """Return formatter CI configuration failures."""
    failures: list[str] = []
    workflow = (repository / ".github/workflows/formatter-ci.yml").read_text()
    required_fragments = (
        "  pull_request:\n",
        "python3 scripts/formatter_ci.py classify",
        "if: needs.changes.outputs.formatter == 'true'",
        "run: make formatter-go-corpus",
        "formatter-corpus-gate:\n"
        "    name: formatter-corpus\n"
        "    needs: [changes, formatter-go-corpus]\n"
        "    if: always()",
        "python3 scripts/formatter_ci.py gate",
    )
    for fragment in required_fragments:
        if fragment not in workflow:
            failures.append(f"formatter workflow needs {fragment.strip()!r}")
    if "\n    paths:" in workflow:
        failures.append("formatter workflow must not use a pull request path filter")

    makefile = (repository / "Makefile").read_text()
    if "formatter-go-corpus:" not in makefile:
        failures.append("Makefile has no formatter-go-corpus target")
    if "slow-ci-unlocked: tgolint-unit-test formatter-go-corpus" not in makefile:
        failures.append("slow CI does not include the formatter corpus")

    slow_workflow = (repository / ".github/workflows/slow-ci.yml").read_text()
    if "schedule:" not in slow_workflow or "run: make slow-ci" not in slow_workflow:
        failures.append("weekly slow CI does not include the formatter corpus")

    for package in local_formatter_packages(repository):
        sample = (package / "dependency.go").as_posix()
        if not affects_formatter([sample]):
            failures.append(
                "formatter classification misses local dependency "
                f"{package.as_posix()}"
            )
    return failures


def classify() -> None:
    """Write the GitHub Actions output for the current pull request."""
    base = os.environ.get("BASE_SHA", "")
    head = os.environ.get("HEAD_SHA", "")
    if not base or not head:
        raise SystemExit("BASE_SHA and HEAD_SHA are required")
    required = affects_formatter(changed_paths(ROOT, base, head))
    print(f"formatter={str(required).lower()}")


def check_gate() -> None:
    """Fail unless the jobs prove the required formatter corpus result."""
    failure = gate_failure(
        os.environ.get("CLASSIFIER_RESULT", ""),
        os.environ.get("FORMATTER_REQUIRED", ""),
        os.environ.get("CORPUS_RESULT", ""),
    )
    if failure is not None:
        raise SystemExit(failure)


def verify() -> None:
    """Fail when repository configuration can bypass the formatter corpus."""
    failures = repository_failures(ROOT)
    if failures:
        raise SystemExit("\n".join(failures))


def main() -> None:
    """Run one formatter CI operation."""
    if len(sys.argv) != 2:
        raise SystemExit("usage: formatter_ci.py {classify|gate|verify}")
    command = sys.argv[1]
    if command == "classify":
        classify()
    elif command == "gate":
        check_gate()
    elif command == "verify":
        verify()
    else:
        raise SystemExit(f"unknown formatter CI operation: {command}")


if __name__ == "__main__":
    main()
