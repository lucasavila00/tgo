#!/usr/bin/env python3
"""Check temporary architecture proposals."""

from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[1]
ADR_DIRECTORY = ROOT / "docs" / "adr"
PERMANENT_FILES = {"README.md", "template.md"}
MAX_PROSE_WIDTH = 80
MAX_LINES = 100
ISSUE_REFERENCE = re.compile(r"(?:#\d+\b|/issues/\d+\b)")
PLACEHOLDER = re.compile(r"<([A-Za-z][^>\n]*)>")
STATUS_FIELD = re.compile(r"^\s*Status\s*:", re.IGNORECASE | re.MULTILINE)


def prose_width_failures(path: Path, content: str) -> list[str]:
    """Return width failures outside fenced code blocks."""
    failures = []
    fence = None
    for number, line in enumerate(content.splitlines(), 1):
        stripped = line.lstrip()
        marker = stripped[:3]
        if marker in {"```", "~~~"}:
            fence = None if fence == marker else marker
            continue
        if fence is None:
            width = len(line.expandtabs(4))
            if width > MAX_PROSE_WIDTH:
                failures.append(
                    f"{path.relative_to(ROOT)}:{number}: "
                    f"{width} characters; ADR prose limit is {MAX_PROSE_WIDTH}"
                )
    return failures


def placeholder(content: str) -> bool:
    """Report an instruction placeholder but allow Markdown URL autolinks."""
    for match in PLACEHOLDER.finditer(content):
        value = match.group(1).lower()
        if not value.startswith(("http://", "https://", "mailto:")):
            return True
    return False


def check(path: Path) -> list[str]:
    """Return all ADR rule failures for one file."""
    content = path.read_text(encoding="utf-8")
    relative = path.relative_to(ROOT)
    failures = prose_width_failures(path, content)
    lines = len(content.splitlines())
    if lines > MAX_LINES:
        failures.append(f"{relative}: {lines} lines; ADR maximum is {MAX_LINES}")
    if STATUS_FIELD.search(content):
        failures.append(f"{relative}: remove the Status field")
    if path.name not in PERMANENT_FILES:
        if placeholder(content):
            failures.append(f"{relative}: remove template instruction placeholders")
        if not ISSUE_REFERENCE.search(content):
            failures.append(f"{relative}: add an issue reference")
    return failures


def main() -> int:
    """Check every Markdown file in the ADR directory."""
    failures = []
    for path in sorted(ADR_DIRECTORY.glob("*.md")):
        failures.extend(check(path))
    for failure in failures:
        print(failure)
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
