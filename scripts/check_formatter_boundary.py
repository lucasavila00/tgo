#!/usr/bin/env python3
"""Keep the TGo formatter independent from Go formatting tools."""

from pathlib import Path
import re
import subprocess


ROOT = Path(__file__).resolve().parents[1]
FORMATTER_DIRS = (ROOT / "pkg" / "format", ROOT / "cmd" / "tgofmt")
GO_HEADER = (
    "// Copyright 2009 The Go Authors. All rights reserved.\n"
    "// Use of this source code is governed by a BSD-style\n"
    "// license that can be found in ../../third_party/go/LICENSE.\n"
)
PROVENANCE = {
    "pkg/format/alignment.tgo": ("math.go", "nodes.go"),
    "pkg/format/binary.tgo": ("nodes.go",),
    "pkg/format/declarations.tgo": ("nodes.go",),
    "pkg/format/expression_lists.tgo": ("nodes.go",),
    "pkg/format/expressions.tgo": ("nodes.go",),
    "pkg/format/go_printer_layout.tgo": ("printer.go",),
    "pkg/format/list_comment_alignment.tgo": ("nodes.go",),
    "pkg/format/printer.tgo": (
        "comment.go", "gobuild.go", "nodes.go", "printer.go",
    ),
    "pkg/format/statements.tgo": ("nodes.go",),
}
PROHIBITED = {
    "go/format": re.compile(r"\bgo/format\b"),
    "go/printer": re.compile(r"\bgo/printer\b"),
    "os/exec": re.compile(r"\bos/exec\b"),
    "gofmt command": re.compile(r"\bgofmt\b"),
}


def main() -> None:
    violations: list[str] = []
    documentation = (ROOT / "docs" / "implementation" / "go-printer-port.md").read_text()
    for name, sources in PROVENANCE.items():
        path = ROOT / name
        if not path.is_file():
            violations.append(f"formatter provenance names missing file {name}")
            continue
        if not path.read_text().startswith(GO_HEADER):
            violations.append(f"{name}: adapted Go source has no Go copyright header")
        source_list = ", ".join(f"`{source}`" for source in sources)
        row = f"| `{name}` | {source_list} |"
        if row not in documentation:
            violations.append(f"{name}: formatter provenance is absent from documentation")

    for path in sorted((ROOT / "pkg" / "format").glob("*.tgo")):
        name = path.relative_to(ROOT).as_posix()
        if path.read_text().startswith(GO_HEADER) and name not in PROVENANCE:
            violations.append(f"{name}: Go copyright header has no provenance entry")

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
    for dependency in ("go/format", "go/printer", "os/exec"):
        if dependency in dependencies:
            violations.append(
                f"formatter dependency closure contains prohibited {dependency}"
            )

    if violations:
        raise SystemExit("formatter crosses its TGo boundary:\n" + "\n".join(violations))


if __name__ == "__main__":
    main()
