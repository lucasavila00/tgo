#!/usr/bin/env python3
"""Check attribution and source maps for adapted upstream code."""

from dataclasses import dataclass
from pathlib import Path
import re


ROOT = Path(__file__).resolve().parents[1]
SOURCE_ROOTS = ("cmd", "internal", "pkg")
GO_COPYRIGHT = re.compile(r"\A// Copyright \d{4} The Go Authors\.")
GENERATED_GO = re.compile(r"_tgo(?:_[^/]+)?\.go$")


@dataclass(frozen=True)
class Adaptation:
    """Describe one local file adapted from an upstream Go source."""

    year: int
    license_path: str
    documentation: str
    sources: tuple[str, ...]


FORMATTER_DOCUMENTATION = "docs/implementation/go-printer-port.md"
CFG_DOCUMENTATION = "docs/implementation/cfg-port.md"
ADAPTATIONS = {
    "pkg/format/alignment.tgo": Adaptation(
        2009, "../../third_party/go/LICENSE", FORMATTER_DOCUMENTATION,
        ("math.go", "nodes.go"),
    ),
    "pkg/format/binary.tgo": Adaptation(
        2009, "../../third_party/go/LICENSE", FORMATTER_DOCUMENTATION,
        ("nodes.go",),
    ),
    "pkg/format/declarations.tgo": Adaptation(
        2009, "../../third_party/go/LICENSE", FORMATTER_DOCUMENTATION,
        ("nodes.go",),
    ),
    "pkg/format/expression_lists.tgo": Adaptation(
        2009, "../../third_party/go/LICENSE", FORMATTER_DOCUMENTATION,
        ("nodes.go",),
    ),
    "pkg/format/expressions.tgo": Adaptation(
        2009, "../../third_party/go/LICENSE", FORMATTER_DOCUMENTATION,
        ("nodes.go",),
    ),
    "pkg/format/go_printer_layout.tgo": Adaptation(
        2009, "../../third_party/go/LICENSE", FORMATTER_DOCUMENTATION,
        ("printer.go",),
    ),
    "pkg/format/list_comment_alignment.tgo": Adaptation(
        2009, "../../third_party/go/LICENSE", FORMATTER_DOCUMENTATION,
        ("nodes.go",),
    ),
    "pkg/format/printer.tgo": Adaptation(
        2009, "../../third_party/go/LICENSE", FORMATTER_DOCUMENTATION,
        ("comment.go", "gobuild.go", "nodes.go", "printer.go"),
    ),
    "pkg/format/statements.tgo": Adaptation(
        2009, "../../third_party/go/LICENSE", FORMATTER_DOCUMENTATION,
        ("nodes.go",),
    ),
    "pkg/syntax/cfg/cfg.go": Adaptation(
        2016, "../../../third_party/go/LICENSE", CFG_DOCUMENTATION,
        ("go/cfg/cfg.go",),
    ),
    "pkg/syntax/cfg/builder.tgo": Adaptation(
        2016, "../../../third_party/go/LICENSE", CFG_DOCUMENTATION,
        ("go/cfg/builder.go",),
    ),
}


def header(adaptation: Adaptation) -> str:
    """Return the required Go Authors header."""
    return (
        f"// Copyright {adaptation.year} The Go Authors. All rights reserved.\n"
        "// Use of this source code is governed by a BSD-style\n"
        f"// license that can be found in {adaptation.license_path}.\n"
    )


def production_sources() -> list[Path]:
    """Return handwritten production Go and TGo source files."""
    result: list[Path] = []
    for source_root in SOURCE_ROOTS:
        for suffix in ("*.go", "*.tgo"):
            for path in (ROOT / source_root).rglob(suffix):
                if "testdata" in path.parts or path.name.endswith(
                    ("_test.go", "_test.tgo")
                ) or GENERATED_GO.search(path.name):
                    continue
                result.append(path)
    return sorted(set(result))


def main() -> None:
    """Report missing attribution or source-map entries."""
    violations: list[str] = []
    documentation: dict[str, str] = {}
    for name, adaptation in ADAPTATIONS.items():
        path = ROOT / name
        if not path.is_file():
            violations.append(f"upstream provenance names missing file {name}")
            continue
        if not path.read_text().startswith(header(adaptation)):
            violations.append(f"{name}: adapted source has no required header")
        content = documentation.setdefault(
            adaptation.documentation,
            (ROOT / adaptation.documentation).read_text(),
        )
        source_list = ", ".join(f"`{source}`" for source in adaptation.sources)
        row = f"| `{name}` | {source_list} |"
        if row not in content:
            violations.append(f"{name}: source map is absent from documentation")

    for path in production_sources():
        name = path.relative_to(ROOT).as_posix()
        if GO_COPYRIGHT.match(path.read_text()) and name not in ADAPTATIONS:
            violations.append(f"{name}: Go copyright header has no source map")

    if violations:
        raise SystemExit("upstream provenance is incomplete:\n" + "\n".join(violations))


if __name__ == "__main__":
    main()
