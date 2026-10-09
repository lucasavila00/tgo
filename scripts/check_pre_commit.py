#!/usr/bin/env python3
"""Keep the pre-commit hook small and portable."""

from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
HOOK = ROOT / ".githooks" / "pre-commit"
EXPECTED = """#!/bin/sh
set -eu
cd "$(git rev-parse --show-toplevel)"
git diff --cached --check
TGOFMT_STAGED=1 make tgofmt-check
MARKDOWN_STAGED=1 exec make markdown
"""


def main() -> None:
    """Reject changes that add broad work to the commit hook."""
    if HOOK.read_text() != EXPECTED:
        raise SystemExit(
            ".githooks/pre-commit must run only staged whitespace, TGo format, and Markdown checks"
        )


if __name__ == "__main__":
    main()
