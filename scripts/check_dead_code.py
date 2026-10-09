#!/usr/bin/env python3
"""Reject unreachable repository functions."""

from __future__ import annotations

import json
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
EXCLUSIONS = {
    ("internal/driver/driver.tgo", "CompileWorkspace"):
        "public Go API for callers that do not need a context",
    ("pkg/syntax/walk.tgo", "ExtensionAt"):
        "public syntax API for external source tools",
    ("pkg/syntax/walk.tgo", "AttachedComments"):
        "public syntax API for external source tools",
}


@dataclass(frozen=True)
class Finding:
    """One deadcode result with its source owner."""

    name: str
    kind: str
    file: str
    line: int
    generated_file: str | None = None
    generated_line: int | None = None


def tgo_owner(path: str) -> str | None:
    """Return the TGo source name for generated Go output."""
    match = re.fullmatch(r"(.+)_tgo((?:_[^/]+)?)\.go", path)
    if match is None:
        return None
    return f"{match.group(1)}{match.group(2)}.tgo"


def declaration_line(source: Path, name: str) -> int | None:
    """Find a named function or method in one TGo source file."""
    receiver, separator, short_name = name.rpartition(".")
    function = re.compile(
        rf"^\s*func\s+{re.escape(short_name)}(?:\s*\[|\s*\()"
    )
    method = re.compile(
        rf"^\s*func\s+\(([^)]*)\)\s*{re.escape(short_name)}(?:\s*\[|\s*\()"
    )
    receiver_type = re.compile(
        r"(?:^|\s)[%*]?([A-Za-z_]\w*)(?:\[[^]]+\])?\s*$"
    )
    for line, text in enumerate(source.read_text().splitlines(), 1):
        if not separator and function.search(text):
            return line
        found = method.search(text)
        if separator and found is not None:
            found_receiver = receiver_type.search(found.group(1))
            if found_receiver is not None and found_receiver.group(1) == receiver:
                return line
    return None


def type_line(source: Path, name: str) -> int | None:
    """Find one named type in a TGo source file."""
    pattern = re.compile(rf"^\s*type\s+{re.escape(name)}(?:\s|\[)")
    for line, text in enumerate(source.read_text().splitlines(), 1):
        if pattern.search(text):
            return line
    return None


def generated_protocol_function(name: str) -> bool:
    """Identify required compiler-generated model protocol functions."""
    method = name.rsplit(".", 1)[-1]
    if method in {
        "Tag", "UnknownTag", "GobEncode", "GobDecode",
        "MarshalJSON", "MarshalJSONTo", "UnmarshalJSON", "UnmarshalJSONFrom",
    }:
        return True
    return method.endswith("Payload") or bool(
        re.fullmatch(r"tgo.+ExternalJSONTo", name)
    )


def source_finding(item: dict[str, object], root: Path) -> Finding | None:
    """Map a deadcode item to handwritten Go or its owning TGo declaration."""
    position = item["Position"]
    if not isinstance(position, dict):
        raise ValueError("deadcode returned an invalid position")
    file = str(position["File"])
    line = int(position["Line"])
    name = str(item["Name"])
    if not bool(item["Generated"]):
        return Finding(name, "func", file, line)

    owner = tgo_owner(file)
    if owner is None or not (root / owner).is_file():
        return Finding(name, "func", file, line, file, line)
    owner_line = declaration_line(root / owner, name)
    if owner_line is not None:
        return Finding(name, "func", owner, owner_line, file, line)
    if generated_protocol_function(name):
        # These functions form the model ABI. A source author cannot remove one.
        return None
    receiver = name.split(".", 1)[0] if "." in name else ""
    owner_line = type_line(root / owner, receiver) if receiver else None
    return Finding(name, "func", owner, owner_line or 1, file, line)


def source_declaration_line(source: Path, name: str, kind: str) -> int | None:
    """Find one package-level type, variable, or constant in TGo source."""
    if kind == "type":
        return type_line(source, name)
    direct = re.compile(rf"^{kind}\s+[^/=]*\b{re.escape(name)}\b")
    grouped = False
    for line, text in enumerate(source.read_text().splitlines(), 1):
        if re.match(rf"^{kind}\s*\(", text):
            grouped = True
            continue
        if grouped and re.match(r"^\s*\)", text):
            grouped = False
            continue
        grouped_names = text.split("=", 1)[0] if grouped else ""
        if direct.search(text) or re.search(
            rf"\b{re.escape(name)}\b", grouped_names
        ):
            return line
    return None


def declaration_finding(item: dict[str, object], root: Path) -> Finding:
    """Map one dead declaration to Go source or its owning TGo declaration."""
    position = item["Position"]
    if not isinstance(position, dict):
        raise ValueError("declaration analysis returned an invalid position")
    file = str(position["File"])
    line = int(position["Line"])
    name = str(item["Name"])
    kind = str(item["Kind"])
    if not bool(item["Generated"]):
        return Finding(name, kind, file, line)

    owner = tgo_owner(file)
    if owner is None or not (root / owner).is_file():
        return Finding(name, kind, file, line, file, line)
    owner_line = source_declaration_line(root / owner, name, kind)
    if owner_line is None:
        return Finding(name, kind, file, line, file, line)
    return Finding(name, kind, owner, owner_line, file, line)


def check(
    records: list[dict[str, object]],
    declarations: list[dict[str, object]],
    root: Path,
    exclusions: dict[tuple[str, str], str],
) -> list[str]:
    """Return stable diagnostics and reject stale exclusions."""
    for key, reason in exclusions.items():
        if not reason.strip():
            raise ValueError(f"dead-code exclusion has no reason: {key[0]}: {key[1]}")
    diagnostics = []
    used_exclusions = set()
    findings = []
    for package in records:
        functions = package.get("Funcs", [])
        if not isinstance(functions, list):
            raise ValueError("deadcode returned an invalid function list")
        for item in functions:
            if not isinstance(item, dict):
                raise ValueError("deadcode returned an invalid function")
            finding = source_finding(item, root)
            if finding is not None:
                findings.append(finding)
    for item in declarations:
        findings.append(declaration_finding(item, root))

    for finding in sorted(
        findings, key=lambda item: (item.file, item.line, item.kind, item.name)
    ):
        key = (finding.file, finding.name)
        if key in exclusions:
            used_exclusions.add(key)
            continue
        kind = "Go source"
        if finding.generated_file is not None:
            kind = (
                "TGo source; generated at "
                f"{finding.generated_file}:{finding.generated_line}"
            )
        diagnostics.append(
            f"{finding.file}:{finding.line}: unreachable {finding.kind}: "
            f"{finding.name} ({kind})"
        )

    for key in sorted(set(exclusions) - used_exclusions):
        diagnostics.append(f"stale dead-code exclusion: {key[0]}: {key[1]}")
    return diagnostics


def deadcode(packages: list[str]) -> list[dict[str, object]]:
    """Run the pinned whole-program analyzer."""
    command = [
        "go", "tool", "deadcode",
        "-test", "-generated", "-json", *packages,
    ]
    result = subprocess.run(
        command,
        cwd=ROOT,
        check=True,
        stdout=subprocess.PIPE,
        text=True,
    )
    records = json.loads(result.stdout)
    return [] if records is None else records


def dead_declarations(
    records: list[dict[str, object]], packages: list[str]
) -> list[dict[str, object]]:
    """Find unreachable package-level types, variables, and constants."""
    result = subprocess.run(
        ["go", "run", "./scripts/deaddecl", *packages],
        cwd=ROOT,
        check=True,
        input=json.dumps(records),
        stdout=subprocess.PIPE,
        text=True,
    )
    declarations = json.loads(result.stdout)
    return [] if declarations is None else declarations


def main() -> int:
    """Check all repository packages and commands."""
    packages = ["./..."]
    records = deadcode(packages)
    declarations = dead_declarations(records, packages)
    diagnostics = check(records, declarations, ROOT, EXCLUSIONS)
    for diagnostic in diagnostics:
        print(diagnostic)
    return 1 if diagnostics else 0


if __name__ == "__main__":
    sys.exit(main())
