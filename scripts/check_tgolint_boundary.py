#!/usr/bin/env python3
"""Keep tgolint syntax work on the TGo syntax and type adapters."""

from pathlib import Path
import re


ROOT = Path(__file__).resolve().parents[1]
TGOLINT = ROOT / "internal" / "tgolint"
TYPE_ADAPTER = Path("internal/tgolint/go_types.tgo")
TYPE_CHECKER_IMPORTS = {
    "go/constant",
    "go/token",
    "go/types",
}
SOURCE_IMPORTS = {
    "go/format": Path("internal/tgolint/source_models.tgo"),
}
IMPORT = re.compile(r'"(go/[^"]+)"')
SYNTAX_ASSERTION = re.compile(r"\.\(\*?syntax\.")


def main() -> None:
    violations: list[str] = []
    for path in sorted(TGOLINT.glob("*.tgo")):
        relative = path.relative_to(ROOT)
        source = path.read_text()
        for package in IMPORT.findall(source):
            if package in TYPE_CHECKER_IMPORTS:
                continue
            if SOURCE_IMPORTS.get(package) != relative:
                violations.append(f"{relative}: prohibited import {package}")
        if ".(type)" in source and relative != TYPE_ADAPTER:
            violations.append(f"{relative}: Go type switch must use the TGo type adapter")
        if SYNTAX_ASSERTION.search(source):
            violations.append(f"{relative}: use pkg/syntax enum accessors")

    for path in sorted(TGOLINT.glob("*.go")):
        if not path.name.endswith(("_test.go", "_tgo.go")):
            relative = path.relative_to(ROOT)
            violations.append(f"{relative}: production tgolint source must be TGo")

    if violations:
        raise SystemExit("tgolint crosses its TGo boundary:\n" + "\n".join(violations))


if __name__ == "__main__":
    main()
