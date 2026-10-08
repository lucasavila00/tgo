#!/usr/bin/env python3
"""Run allocation benchmarks and enforce their memory budgets."""

from __future__ import annotations

import json
import re
import shutil
import subprocess
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
CONFIG = Path(__file__).with_name("budgets.json")
RESULT = re.compile(
    r"^(Benchmark\S+)-\d+\s+\d+\s+"
    r"(?:\d+(?:\.\d+)?\s+ns/op\s+)?"
    r"(\d+)\s+B/op\s+(\d+)\s+allocs/op$"
)


def benchmark(config: dict, directory: Path) -> str:
    """Run the configured benchmarks and return their output."""
    command = [
        "go",
        "test",
        config["package"],
        "-run",
        "^$",
        "-bench",
        "^BenchmarkEnumJSON",
        "-benchmem",
        "-benchtime=100000x",
        "-count=1",
    ]
    result = subprocess.run(command, cwd=directory, text=True, capture_output=True)
    print(result.stdout, end="")
    print(result.stderr, end="")
    if result.returncode != 0:
        raise SystemExit(result.returncode)
    return result.stdout


def generated_fixture(config: dict, temporary: Path) -> Path:
    """Compile a private fixture copy with the current compiler."""
    compiler = temporary / "tgo"
    subprocess.run(
        ["go", "build", "-o", str(compiler), "./cmd/tgo"],
        cwd=ROOT,
        check=True,
    )
    source = ROOT / config["directory"]
    fixture = temporary / "fixture"
    shutil.copytree(source, fixture)
    subprocess.run([str(compiler), "build", "./..."], cwd=fixture, check=True)
    return fixture


def main() -> None:
    """Run each benchmark once and reject memory budget increases."""
    config = json.loads(CONFIG.read_text())
    with tempfile.TemporaryDirectory(prefix="tgo-allocations-") as path:
        output = benchmark(config, generated_fixture(config, Path(path)))

    actual = {}
    for line in output.splitlines():
        match = RESULT.match(line)
        if match:
            actual[match.group(1)] = {
                "bytes": int(match.group(2)),
                "allocations": int(match.group(3)),
            }

    expected = config["benchmarks"]
    missing = sorted(set(expected) - set(actual))
    extra = sorted(set(actual) - set(expected))
    failures = []
    if missing:
        failures.append("missing benchmarks: " + ", ".join(missing))
    if extra:
        failures.append("unbudgeted benchmarks: " + ", ".join(extra))
    for name, budget in expected.items():
        if name not in actual:
            continue
        for metric in ("bytes", "allocations"):
            if actual[name][metric] > budget[metric]:
                failures.append(
                    f"{name}: {actual[name][metric]} {metric}; "
                    f"maximum is {budget[metric]}"
                )

    if failures:
        raise SystemExit("\n".join(failures))
    print(f"PASS {len(expected)} allocation budgets")


if __name__ == "__main__":
    main()
