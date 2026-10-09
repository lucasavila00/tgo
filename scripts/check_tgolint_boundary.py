#!/usr/bin/env python3
"""Keep tgolint syntax work on the TGo syntax and type adapters."""

from pathlib import Path
import re


ROOT = Path(__file__).resolve().parents[1]
TGOLINT = ROOT / "internal" / "tgolint"
TYPE_ADAPTER = Path("internal/tgolint/go_types.tgo")
ALLOWED_GO_IMPORTS = {
    "go/constant",
    "go/format",
    "go/scanner",
    "go/token",
    "go/types",
}
IMPORT = re.compile(r'"(go/[^"]+)"')


def main() -> None:
    violations: list[str] = []
    for path in sorted(TGOLINT.glob("*.tgo")):
        relative = path.relative_to(ROOT)
        source = path.read_text()
        for package in IMPORT.findall(source):
            if package not in ALLOWED_GO_IMPORTS:
                violations.append(f"{relative}: prohibited import {package}")
        if ".(type)" in source and relative != TYPE_ADAPTER:
            violations.append(f"{relative}: Go type switch must use the TGo type adapter")

    if violations:
        raise SystemExit("tgolint crosses its TGo boundary:\n" + "\n".join(violations))


if __name__ == "__main__":
    main()
