#!/usr/bin/env python3
"""Reject go/ast use outside parser and compiler implementation packages."""

from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
ALLOWED = (Path("internal/compiler"), Path("pkg/syntax"))


def main() -> None:
    violations = []
    for suffix in ("*.go", "*.tgo"):
        for path in ROOT.rglob(suffix):
            relative = path.relative_to(ROOT)
            if any(relative.is_relative_to(directory) for directory in ALLOWED):
                continue
            if '"go/ast"' in path.read_text():
                violations.append(relative)
    if violations:
        paths = "\n".join(str(path) for path in sorted(set(violations)))
        raise SystemExit(f"go/ast crosses the downstream syntax boundary:\n{paths}")


if __name__ == "__main__":
    main()
