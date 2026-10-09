#!/usr/bin/env python3
"""Keep the TGo formatter independent from Go formatting tools."""

from pathlib import Path
import re
import subprocess


ROOT = Path(__file__).resolve().parents[1]
FORMATTER_DIRS = (ROOT / "pkg" / "format", ROOT / "cmd" / "tgofmt")
PROHIBITED = {
    "go/format": re.compile(r"\bgo/format\b"),
    "go/printer": re.compile(r"\bgo/printer\b"),
    "gofmt command": re.compile(r"\bgofmt\b"),
}
OS_EXEC = re.compile(r"\bos/exec\b")


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

    dependencies = subprocess.run(
        [
            "go",
            "list",
            "-deps",
            "-f",
            "{{.ImportPath}}",
            "./pkg/format",
            "./cmd/tgofmt",
        ],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    ).stdout.splitlines()
    for dependency in ("go/format", "go/printer"):
        if dependency in dependencies:
            violations.append(
                f"formatter dependency closure contains prohibited {dependency}"
            )

    module = subprocess.run(
        ["go", "list", "-m", "-f", "{{.Path}}"],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()
    for dependency in dependencies:
        if dependency == module:
            directory = ROOT
        elif dependency.startswith(module + "/"):
            directory = ROOT / dependency.removeprefix(module + "/")
        else:
            continue
        for path in sorted(directory.iterdir()):
            if path.suffix not in {".go", ".tgo"} or path.name.endswith(
                ("_test.go", "_test.tgo")
            ):
                continue
            if OS_EXEC.search(path.read_text()):
                relative = path.relative_to(ROOT)
                violations.append(f"{relative}: prohibited os/exec reference")

    if violations:
        raise SystemExit("formatter crosses its TGo boundary:\n" + "\n".join(violations))


if __name__ == "__main__":
    main()
