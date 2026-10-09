#!/usr/bin/env python3
"""Check temporary architecture proposals."""

from __future__ import annotations

from pathlib import Path
import re


ROOT = Path(__file__).resolve().parents[1]
ADR_DIRECTORY = ROOT / "docs" / "adr"
PERMANENT_FILES = frozenset({"README.md", "template.md"})
MAX_PROSE_WIDTH = 80
MAX_LINES = 100
ISSUE_REFERENCE = re.compile(r"(?:#\d+\b|/issues/\d+\b)")
PLACEHOLDER = re.compile(r"<([A-Za-z][^>\n]*)>")
STATUS_FIELD = re.compile(r"^\s*Status\s*:", re.IGNORECASE | re.MULTILINE)


def check_content(name: str, content: str, proposal: bool) -> list[str]:
    """Return all ADR rule failures for one file."""
    failures: list[str] = []
    fence: str | None = None
    for number, line in enumerate(content.splitlines(), 1):
        marker = line.lstrip()[:3]
        if marker in {"```", "~~~"}:
            fence = None if fence == marker else marker
            continue
        width = len(line.expandtabs(4))
        if fence is None and width > MAX_PROSE_WIDTH:
            failures.append(
                f"{name}:{number}: {width} characters; "
                f"ADR prose limit is {MAX_PROSE_WIDTH}"
            )
    lines = len(content.splitlines())
    if lines > MAX_LINES:
        failures.append(f"{name}: {lines} lines; ADR maximum is {MAX_LINES}")
    if STATUS_FIELD.search(content):
        failures.append(f"{name}: remove the Status field")
    if proposal:
        for match in PLACEHOLDER.finditer(content):
            if not match.group(1).lower().startswith(
                ("http://", "https://", "mailto:")
            ):
                failures.append(f"{name}: remove template instruction placeholders")
                break
        if not ISSUE_REFERENCE.search(content):
            failures.append(f"{name}: add an issue reference")
    return failures


def check_directory(directory: Path) -> list[str]:
    """Return failures for every Markdown file in an ADR directory."""
    failures: list[str] = []
    for path in sorted(directory.glob("*.md")):
        failures.extend(
            check_content(
                str(path.relative_to(ROOT)),
                path.read_text(encoding="utf-8"),
                path.name not in PERMANENT_FILES,
            )
        )
    return failures


def main() -> None:
    """Report ADR rule failures."""
    failures = check_directory(ADR_DIRECTORY)
    if failures:
        raise SystemExit("\n".join(failures))


if __name__ == "__main__":
    main()
