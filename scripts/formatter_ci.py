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
        ".github/workflows/ci.yml",
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
    corpus_passed: str,
) -> str | None:
    """Return an error when job results do not prove the required work."""
    if classifier_result != "success":
        return f"formatter change classification ended with {classifier_result}"
    if formatter_required == "true":
        if corpus_result != "success":
            return f"required formatter corpus ended with {corpus_result}"
        if corpus_passed != "true":
            return "required formatter corpus did not prove success"
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


def workflow_section(source: str, header: str) -> list[str] | None:
    """Return lines nested below one exact YAML mapping key."""
    lines = source.splitlines()
    try:
        start = lines.index(header)
    except ValueError:
        return None
    indentation = len(header) - len(header.lstrip())
    section: list[str] = []
    for line in lines[start + 1 :]:
        if line.strip() and len(line) - len(line.lstrip()) <= indentation:
            break
        section.append(line)
    return section


def active_lines(lines: Iterable[str]) -> list[str]:
    """Return nonblank, uncommented configuration lines."""
    return [
        line
        for line in lines
        if line.strip() and not line.lstrip().startswith("#")
    ]


def has_active_key(lines: Iterable[str], key: str) -> bool:
    """Report whether active configuration contains one mapping key."""
    prefix = key + ":"
    return any(line.lstrip().startswith(prefix) for line in active_lines(lines))


def formatter_workflow_failures(source: str) -> list[str]:
    """Return failures in active formatter workflow sections."""
    failures: list[str] = []
    event = workflow_section(source, "on:")
    pull_request = (
        None
        if event is None
        else workflow_section("\n".join(event), "  pull_request:")
    )
    if pull_request is None:
        failures.append("formatter workflow needs an active pull_request event")
    elif active_lines(pull_request):
        failures.append("formatter pull_request event must not have options")

    jobs = workflow_section(source, "jobs:")
    required_jobs = {
        "  changes:": (
            "      formatter: ${{ steps.classify.outputs.formatter }}",
            "          fetch-depth: 0",
            "          BASE_SHA: ${{ github.event.pull_request.base.sha }}",
            "          HEAD_SHA: ${{ github.event.pull_request.head.sha }}",
            '        run: python3 scripts/formatter_ci.py classify >> "$GITHUB_OUTPUT"',
        ),
        "  formatter-go-corpus:": (
            "    needs: changes",
            "    if: needs.changes.outputs.formatter == 'true'",
            "      passed: ${{ steps.corpus.outputs.passed }}",
            "      - id: corpus",
        ),
        "  formatter-corpus-gate:": (
            "    name: formatter-corpus",
            "    needs: [changes, formatter-go-corpus]",
            "    if: always()",
            "          CLASSIFIER_RESULT: ${{ needs.changes.result }}",
            "          FORMATTER_REQUIRED: ${{ needs.changes.outputs.formatter }}",
            "          CORPUS_RESULT: ${{ needs.formatter-go-corpus.result }}",
            "          CORPUS_PASSED: ${{ needs.formatter-go-corpus.outputs.passed }}",
            "        run: python3 scripts/formatter_ci.py gate",
        ),
    }
    if jobs is None:
        failures.append("formatter workflow needs an active jobs section")
        return failures
    jobs_source = "\n".join(jobs)
    job_sections: dict[str, list[str]] = {}
    for header, required_lines in required_jobs.items():
        job = workflow_section(jobs_source, header)
        name = header.strip().removesuffix(":")
        if job is None:
            failures.append(f"formatter workflow needs an active {name} job")
            continue
        job_sections[header] = job
        for line in required_lines:
            if line not in job:
                failures.append(f"formatter {name} job needs {line.strip()!r}")

    corpus_job = job_sections.get("  formatter-go-corpus:")
    if corpus_job is not None:
        outputs = workflow_section("\n".join(corpus_job), "    outputs:")
        expected_output = [
            "      passed: ${{ steps.corpus.outputs.passed }}",
        ]
        if outputs is None or active_lines(outputs) != expected_output:
            failures.append("formatter corpus job must export the corpus proof")
        if has_active_key(corpus_job, "continue-on-error"):
            failures.append("formatter corpus job must not continue on error")
        checkout = workflow_section(
            "\n".join(corpus_job),
            "      - uses: actions/checkout@v6",
        )
        checkout_steps = [
            line
            for line in active_lines(corpus_job)
            if line.lstrip().startswith("- uses: actions/checkout@")
        ]
        if (
            checkout is None
            or active_lines(checkout)
            or checkout_steps != ["      - uses: actions/checkout@v6"]
        ):
            failures.append("formatter corpus checkout must not have options")
        corpus_step = workflow_section(
            "\n".join(corpus_job),
            "      - id: corpus",
        )
        if corpus_step is not None:
            if has_active_key(corpus_step, "if"):
                failures.append("formatter corpus step must not have a condition")
            if has_active_key(corpus_step, "shell"):
                failures.append("formatter corpus step must use the default shell")
            expected_step = [
                "        name: Run formatter corpus",
                "        run: make formatter-go-corpus",
            ]
            if active_lines(corpus_step) != expected_step:
                failures.append("formatter corpus step must run only the corpus target")

    gate_job = job_sections.get("  formatter-corpus-gate:")
    if gate_job is not None:
        gate_step = workflow_section(
            "\n".join(gate_job),
            "      - name: Check formatter corpus result",
        )
        expected_environment = [
            "          CLASSIFIER_RESULT: ${{ needs.changes.result }}",
            "          FORMATTER_REQUIRED: ${{ needs.changes.outputs.formatter }}",
            "          CORPUS_RESULT: ${{ needs.formatter-go-corpus.result }}",
            "          CORPUS_PASSED: ${{ needs.formatter-go-corpus.outputs.passed }}",
        ]
        if gate_step is None:
            failures.append("formatter gate needs its result-check step")
        else:
            if has_active_key(gate_step, "if"):
                failures.append("formatter gate step must not have a condition")
            if has_active_key(gate_step, "continue-on-error"):
                failures.append("formatter gate step must not continue on error")
            if has_active_key(gate_step, "shell"):
                failures.append("formatter gate step must use the default shell")
            environment = workflow_section("\n".join(gate_step), "        env:")
            if (
                environment is None
                or active_lines(environment) != expected_environment
            ):
                failures.append("formatter gate must read the exact job results")
            if "        run: python3 scripts/formatter_ci.py gate" not in gate_step:
                failures.append("formatter gate must run its result check")
    return failures


def make_target(source: str, name: str) -> tuple[str, list[str]] | None:
    """Return one active Make target declaration and its recipes."""
    lines = source.splitlines()
    prefix = name + ":"
    matches = [index for index, line in enumerate(lines) if line.startswith(prefix)]
    if len(matches) != 1:
        return None
    index = matches[0]
    recipes: list[str] = []
    for candidate in lines[index + 1 :]:
        if candidate and not candidate[0].isspace():
            break
        if (
            candidate.startswith("\t")
            and candidate.strip()
            and not candidate.lstrip().startswith("#")
        ):
            recipes.append(candidate)
    return lines[index], recipes


def makefile_failures(source: str) -> list[str]:
    """Return failures in the active formatter corpus Make targets."""
    failures: list[str] = []
    slow = make_target(source, "slow-ci-unlocked")
    expected_slow = (
        "slow-ci-unlocked: tgolint-unit-test-unlocked "
        "formatter-go-corpus-unlocked"
    )
    if slow is None or slow[0] != expected_slow:
        failures.append("slow-ci-unlocked must depend on formatter-go-corpus")

    slow_entry = make_target(source, "slow-ci")
    expected_slow_recipe = [
        "\t+@scripts/with-local-validation-lock.sh "
        "$(MAKE) -j2 slow-ci-unlocked",
    ]
    if slow_entry is None or slow_entry[0] != "slow-ci:":
        failures.append("Makefile needs an active slow-ci target")
    elif slow_entry[1] != expected_slow_recipe:
        failures.append("slow-ci must run the exact locked slow CI recipe")

    corpus = make_target(source, "formatter-go-corpus-unlocked")
    expected_recipe = [
        "\tTGO_FULL_GO_FORMAT_CORPUS=1 go test ./pkg/format "
        "-run TestSourceMatchesFullGoTree -count=1 && \\",
        '\t\t{ test -z "$$GITHUB_OUTPUT" || echo "passed=true" '
        '>> "$$GITHUB_OUTPUT"; }',
    ]
    if corpus is None or corpus[0] != "formatter-go-corpus-unlocked:":
        failures.append("Makefile needs an active formatter-go-corpus target")
    elif corpus[1] != expected_recipe:
        failures.append("formatter-go-corpus must run the exact full corpus recipe")
    for line in active_lines(source.splitlines()):
        if not line.startswith(".IGNORE:"):
            continue
        ignored = line.removeprefix(".IGNORE:").split()
        if not ignored or "formatter-go-corpus" in ignored:
            failures.append("Makefile must not ignore formatter-go-corpus failures")
            break
    return failures


def slow_workflow_failures(source: str) -> list[str]:
    """Return failures in the active weekly Slow CI workflow."""
    failures: list[str] = []
    event = workflow_section(source, "on:")
    schedule = (
        None if event is None else workflow_section("\n".join(event), "  schedule:")
    )
    if (
        schedule is None
        or '    - cron: "0 4 * * 1"' not in active_lines(schedule)
    ):
        failures.append("Slow CI needs its active weekly schedule")

    jobs = workflow_section(source, "jobs:")
    tests = None if jobs is None else workflow_section("\n".join(jobs), "  tests:")
    if tests is None:
        failures.append("Slow CI needs its active tests job")
    else:
        if "      - run: make slow-ci" not in active_lines(tests):
            failures.append("Slow CI tests job must run make slow-ci")
        if has_active_key(tests, "if"):
            failures.append("Slow CI tests job must not have a condition")
        if has_active_key(tests, "continue-on-error"):
            failures.append("Slow CI tests job must not continue on error")
    return failures


def repository_failures(repository: Path) -> list[str]:
    """Return formatter CI configuration failures."""
    workflow = (repository / ".github/workflows/formatter-ci.yml").read_text()
    failures = formatter_workflow_failures(workflow)

    makefile = (repository / "Makefile").read_text()
    failures.extend(makefile_failures(makefile))

    slow_workflow = (repository / ".github/workflows/slow-ci.yml").read_text()
    failures.extend(slow_workflow_failures(slow_workflow))

    for package in local_formatter_packages(repository):
        sample = (package / "dependency.go").as_posix()
        if not affects_formatter([sample]):
            failures.append(
                "formatter classification misses local dependency "
                f"{package.as_posix()}"
            )
    return failures


def classify(repository: Path = ROOT) -> None:
    """Write the GitHub Actions output for the current pull request."""
    base = os.environ.get("BASE_SHA", "")
    head = os.environ.get("HEAD_SHA", "")
    if not base or not head:
        raise SystemExit("BASE_SHA and HEAD_SHA are required")
    required = affects_formatter(changed_paths(repository, base, head))
    print(f"formatter={str(required).lower()}")


def check_gate() -> None:
    """Fail unless the jobs prove the required formatter corpus result."""
    failure = gate_failure(
        os.environ.get("CLASSIFIER_RESULT", ""),
        os.environ.get("FORMATTER_REQUIRED", ""),
        os.environ.get("CORPUS_RESULT", ""),
        os.environ.get("CORPUS_PASSED", ""),
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
