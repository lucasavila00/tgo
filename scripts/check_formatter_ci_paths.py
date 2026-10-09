#!/usr/bin/env python3
"""Check pull request coverage for the full formatter corpus."""

from pathlib import Path
import re
import subprocess


ROOT = Path(__file__).resolve().parents[1]
MAKEFILE = ROOT / "Makefile"
WORKFLOW = ROOT / ".github" / "workflows" / "formatter-ci.yml"
SLOW_WORKFLOW = ROOT / ".github" / "workflows" / "slow-ci.yml"
REQUIRED_PATHS = {
    ".github/workflows/formatter-ci.yml": "the formatter corpus workflow",
    ".github/workflows/slow-ci.yml": "the weekly formatter corpus workflow",
    "Makefile": "the formatter corpus command",
    "go.mod": "the Go version and module definition",
    "go.sum": "the module checksums",
    "cmd/tgofmt/**": "the formatter command",
    "pkg/format/**": "the formatter implementation",
    "pkg/syntax/**": "the formatter syntax model",
    "scripts/check_formatter_boundary.py": "the formatter boundary check",
    "scripts/check_formatter_ci_paths.py": "the formatter CI path check",
    "scripts/check_generated.py": "the generated source check",
}
PATH_ENTRY = re.compile(r'^\s{6}-\s+["\'](.+)["\']$')


def pull_request_paths(source: str) -> set[str]:
    """Return the pull request path filters from one workflow."""
    paths: set[str] = set()
    in_pull_request = False
    in_paths = False
    for line in source.splitlines():
        if line == "  pull_request:":
            in_pull_request = True
            continue
        if in_pull_request and line.startswith("  ") and not line.startswith("    "):
            break
        if in_pull_request and line == "    paths:":
            in_paths = True
            continue
        if in_paths:
            match = PATH_ENTRY.fullmatch(line)
            if match:
                paths.add(match.group(1))
            elif line.strip():
                break
    return paths


def local_formatter_packages() -> list[tuple[str, Path]]:
    """Return local packages in the formatter dependency closure."""
    output = subprocess.check_output(
        [
            "go", "list", "-deps", "-f", "{{.ImportPath}}\t{{.Dir}}",
            "./pkg/format", "./cmd/tgofmt",
        ],
        cwd=ROOT,
        text=True,
    )
    packages: list[tuple[str, Path]] = []
    for line in output.splitlines():
        import_path, directory = line.split("\t", maxsplit=1)
        try:
            relative = Path(directory).resolve().relative_to(ROOT)
        except ValueError:
            continue
        packages.append((import_path, relative))
    return packages


def covered(path: Path, patterns: set[str]) -> bool:
    """Report whether a recursive workflow path covers one directory."""
    value = path.as_posix()
    for pattern in patterns:
        if pattern.endswith("/**"):
            root = pattern.removesuffix("/**")
            if value == root or value.startswith(root + "/"):
                return True
    return False


def main() -> None:
    """Check workflow paths, commands, dependencies, and the weekly backstop."""
    workflow = WORKFLOW.read_text()
    paths = pull_request_paths(workflow)
    for path, purpose in REQUIRED_PATHS.items():
        if path not in paths:
            raise SystemExit(f"formatter corpus CI does not cover {purpose}: {path}")

    for import_path, directory in local_formatter_packages():
        if not covered(directory, paths):
            raise SystemExit(
                "formatter corpus CI does not cover local dependency "
                f"{import_path}: {directory.as_posix()}/**"
            )

    if "- run: make formatter-go-corpus" not in workflow:
        raise SystemExit("formatter corpus CI does not run make formatter-go-corpus")

    makefile = MAKEFILE.read_text()
    if "slow-ci-unlocked: tgolint-unit-test formatter-go-corpus" not in makefile:
        raise SystemExit("slow CI no longer includes the formatter corpus")
    slow_workflow = SLOW_WORKFLOW.read_text()
    if "schedule:" not in slow_workflow or "- run: make slow-ci" not in slow_workflow:
        raise SystemExit("weekly slow CI no longer runs the formatter corpus")


if __name__ == "__main__":
    main()
