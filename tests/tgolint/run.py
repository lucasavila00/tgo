#!/usr/bin/env python3
"""Build tgo and tgolint, then check Go boundary diagnostics."""

from pathlib import Path
import shutil
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parents[2]
FIXTURE = Path(__file__).parent / "testdata"


def run(command, cwd, *, success=True):
    result = subprocess.run(command, cwd=cwd, text=True, capture_output=True)
    if (result.returncode == 0) != success:
        raise AssertionError(
            f"{command} exited {result.returncode}\n{result.stdout}{result.stderr}"
        )
    return result


def expected(name):
    return (FIXTURE / name).read_text()


def normalized(text, work):
    return text.replace(str(work) + "/", "")


def main():
    with tempfile.TemporaryDirectory() as temporary:
        temporary = Path(temporary)
        compiler = temporary / "tgo"
        linter = temporary / "tgolint"
        work = temporary / "work"
        shutil.copytree(FIXTURE, work)

        run(["go", "build", "-o", str(compiler), "./cmd/tgo"], ROOT)
        run(["go", "build", "-o", str(linter), "./cmd/tgolint"], ROOT)
        run([str(compiler), "build", "./..."], work)
        run(["go", "test", "./..."], work)

        good = run([str(linter), "./good"], work)
        assert good.stdout == expected("good.stdout"), good.stdout
        assert good.stderr == expected("good.stderr"), good.stderr

        bad = run([str(linter), "./bad"], work, success=False)
        stdout = normalized(bad.stdout, work)
        stderr = normalized(bad.stderr, work)
        assert stdout == expected("bad.stdout"), stdout
        assert stderr == expected("bad.stderr"), stderr

        localbad = run([str(linter), "./localbad"], work, success=False)
        stdout = normalized(localbad.stdout, work)
        stderr = normalized(localbad.stderr, work)
        assert stdout == expected("localbad.stdout"), stdout
        assert stderr == expected("localbad.stderr"), stderr

        genericgood = run([str(linter), "./genericgood"], work)
        assert genericgood.stdout == expected("genericgood.stdout"), genericgood.stdout
        assert genericgood.stderr == expected("genericgood.stderr"), genericgood.stderr

        genericbad = run([str(linter), "./genericbad"], work, success=False)
        stdout = normalized(genericbad.stdout, work)
        stderr = normalized(genericbad.stderr, work)
        assert stdout == expected("genericbad.stdout"), stdout
        assert stderr == expected("genericbad.stderr"), stderr

        genericzerogood = run([str(linter), "./genericzerogood"], work)
        assert genericzerogood.stdout == expected("genericzerogood.stdout")
        assert genericzerogood.stderr == expected("genericzerogood.stderr")

        genericzerobad = run(
            [str(linter), "./genericzerobad"], work, success=False
        )
        stdout = normalized(genericzerobad.stdout, work)
        stderr = normalized(genericzerobad.stderr, work)
        assert stdout == expected("genericzerobad.stdout"), stdout
        assert stderr == expected("genericzerobad.stderr"), stderr


if __name__ == "__main__":
    main()
