#!/usr/bin/env python3
"""Reject active documentation and editor rules for removed TGo syntax."""

from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[1]
ACTIVE_DOCUMENTATION = (
    Path("README.md"),
    Path("docs/spec/README.md"),
    Path("docs/guide/README.md"),
    Path("docs/for-agents/AGENTS.md"),
)
CHECKED_WHERE = re.compile(
    r"\btype\s+[A-Za-z_]\w*\s+[^\n`]*\swhere\s+[A-Za-z0-9_(*&\[]"
)
WHERE_CLAIM = re.compile(r"`where`\s+is\s+contextual", re.IGNORECASE)
WHERE_SCOPE = re.compile(r"keyword\.[^\"\n]*where\.tgo")


def violations(files: dict[str, str]) -> list[str]:
    """Return active claims for the removed checked-where form."""
    failures = []
    for name, source in files.items():
        patterns = (CHECKED_WHERE, WHERE_CLAIM)
        if name.endswith("tgo.tmLanguage.json"):
            patterns += (WHERE_SCOPE,)
        for number, line in enumerate(source.splitlines(), 1):
            if any(pattern.search(line) for pattern in patterns):
                failures.append(
                    f"{name}:{number}: removed checked-where syntax is active"
                )
    return failures


def main() -> int:
    """Check current user documentation and the editor grammar."""
    paths = ACTIVE_DOCUMENTATION + (
        Path("editors/vscode/syntaxes/tgo.tmLanguage.json"),
    )
    files = {path.as_posix(): (ROOT / path).read_text() for path in paths}
    failures = violations(files)
    for failure in failures:
        print(failure)
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
