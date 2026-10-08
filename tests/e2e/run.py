#!/usr/bin/env python3
"""Build the CLI, compile fixture packages, then run their Go tests."""

from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[2]
FIXTURES = Path(__file__).parent / "testdata"


def run(command, cwd, *, success=True):
    result = subprocess.run(command, cwd=cwd, text=True, capture_output=True)
    if (result.returncode == 0) != success:
        raise AssertionError(
            f"{command} exited {result.returncode}\n{result.stdout}{result.stderr}"
        )
    return result.stdout + result.stderr


def check_direct_ffi(work):
    generated = (work / "app" / "ffi_tgo.go").read_text()
    calls = [
        "store.Load(id)",
        "legacy.Map(accounts",
        "legacy.Stream(accounts...)",
        "legacy.Update(account",
        "legacy.TypedNilError()",
        "legacy.Wrap(prefix, values...)",
    ]
    for call in calls:
        assert call in generated, f"generated FFI call is not direct: {call}"


def main():
    with tempfile.TemporaryDirectory(prefix="tgo-e2e-") as temporary:
        temporary = Path(temporary)
        compiler = temporary / "tgo"
        run(["go", "build", "-o", str(compiler), "./cmd/tgo"], ROOT)
        for fixture in sorted(FIXTURES.iterdir()):
            work = temporary / fixture.name
            shutil.copytree(fixture, work)
            committed = {path: path.read_bytes() for path in work.rglob("*_tgo.go")}
            run([str(compiler), "build", "./..."], work)
            for path, expected in committed.items():
                assert path.read_bytes() == expected, f"regenerate committed output: {path}"
            if fixture.name == "business":
                check_direct_ffi(work)
            outputs = {path: path.read_bytes() for path in work.rglob("*_tgo.go")}
            run([str(compiler), "build", "./..."], work)
            assert outputs == {path: path.read_bytes() for path in outputs}
            run(["go", "test", "./..."], work)
            print(f"PASS {fixture.name}")

        invalid = temporary / "invalid"
        invalid.mkdir()
        (invalid / "go.mod").write_text("module invalid.test\n\ngo 1.27.0\n")
        cases = [
            ("var x int", "variables need an initializer"),
            ("type S struct { N int }; var x = S{}", "missing required field N"),
            ("type Q int where value > 0\nvar x = Q{}", "use a constructor for Q"),
            ("type Q int where value > 0\nvar x = new(Q)", "invalid zero value"),
            ("type Q int where value > 0\nvar x = make([]Q, 2)", "constant length 0"),
            (
                "type Q int where value > 0\nfunc(q Q) Break() Q { return Q{} }",
                "use a constructor for Q",
            ),
            (
                "type Q int where value > 0\nfunc f(xs []Q) { clear(xs) }",
                "clear would create invalid slice elements",
            ),
            (
                "type Q int where value > 0\n"
                "func f(xs []Q, n int) []Q { return xs[:n] }",
                "reslice bound must be proven",
            ),
            (
                "type Q int where value > 0\n"
                "func f(xs []Q, n int) []Q { "
                "if n <= len(xs) { n++; return xs[:n] }; return xs }",
                "reslice bound must be proven",
            ),
            (
                "type Q int where value > 0\n"
                "func f(xs map[int]Q) Q { return xs[0] }",
                "this read needs if value, ok",
            ),
            (
                "type Q int where value > 0\n"
                "func f(xs map[int]Q) int { "
                "if value, ok := xs[0]; ok { return value.Value() } "
                "else { return value.Value() } }",
                "value is available only in the successful presence branch",
            ),
            (
                "type Q int where value > 0\n"
                "func f(xs <-chan Q) Q { return <-xs }",
                "this read needs if value, ok",
            ),
            (
                "type Q int where value > 0\n"
                "func f(value any) Q { return value.(Q) }",
                "this read needs if value, ok",
            ),
            ("var x = [2]int{1}", "supply every index"),
            ("var x = []int{2: 1}", "supply every index"),
            (
                "type A enum { One struct {}; Two struct {} }\n"
                "func f(a A) { match a { case One(_): return } }",
                "missing variant Two",
            ),
        ]
        for source, message in cases:
            (invalid / "bad.tgo").write_text("package invalid\n" + source + "\n")
            output = run([str(compiler), "build"], invalid, success=False)
            assert message in output, (message, output)
            assert not list(invalid.glob("*_tgo.go")), "failed build wrote output"
        print(f"PASS {len(cases)} rejected source cases")


if __name__ == "__main__":
    main()
