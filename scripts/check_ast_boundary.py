#!/usr/bin/env python3
"""Reject go/ast use outside parser and compiler implementation packages."""

from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
COMPILER = Path("internal/compiler")
SYNTAX = Path("pkg/syntax")
SYNTAX_PREFIXES = ("convert", "front", "parser_")


def allowed(path: Path) -> bool:
    if path.is_relative_to(COMPILER):
        return True
    return path.parent == SYNTAX and path.name.startswith(SYNTAX_PREFIXES)


def main() -> None:
    violations = []
    for suffix in ("*.go", "*.tgo"):
        for path in ROOT.rglob(suffix):
            relative = path.relative_to(ROOT)
            if allowed(relative):
                continue
            if '"go/ast"' in path.read_text():
                violations.append(relative)
    if violations:
        paths = "\n".join(str(path) for path in sorted(set(violations)))
        raise SystemExit(f"go/ast crosses the downstream syntax boundary:\n{paths}")


if __name__ == "__main__":
    main()
