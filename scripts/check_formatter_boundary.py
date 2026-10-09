#!/usr/bin/env python3
"""Keep the TGo formatter independent from Go formatting tools."""

from pathlib import Path
import re


ROOT = Path(__file__).resolve().parents[1]
FORMATTER_DIRS = (ROOT / "pkg" / "format", ROOT / "cmd" / "tgofmt")
PROHIBITED = {
    "go/format": re.compile(r"\bgo/format\b"),
    "go/printer": re.compile(r"\bgo/printer\b"),
    "os/exec": re.compile(r"\bos/exec\b"),
    "gofmt command": re.compile(r"\bgofmt\b"),
}


def main() -> None:
    violations: list[str] = []
    for directory in FORMATTER_DIRS:
        for path in sorted(directory.rglob("*")):
            if path.suffix not in {".go", ".tgo"}:
                continue
            if "testdata" in path.parts or path.name.endswith(
                ("_test.go", "_test.tgo", "_tgo.go")
            ):
                continue

            relative = path.relative_to(ROOT)
            if path.suffix != ".tgo":
                violations.append(f"{relative}: production formatter source must be TGo")

            source = path.read_text()
            for name, pattern in PROHIBITED.items():
                if pattern.search(source):
                    violations.append(f"{relative}: prohibited {name} reference")

    if violations:
        raise SystemExit("formatter crosses its TGo boundary:\n" + "\n".join(violations))


if __name__ == "__main__":
    main()
