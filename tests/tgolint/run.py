#!/usr/bin/env python3
"""Build tgo and tgolint, then check their diagnostics."""

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


def normalized(text, work):
    return text.replace(str(work) + "/", "")


def fixture_directories():
    for stdout in sorted(FIXTURE.glob("*/stdout")):
        directory = stdout.parent
        if (directory / "stderr").is_file():
            yield directory


def assert_case(linter, work, fixture):
    expected_stdout = (fixture / "stdout").read_text()
    expected_stderr = (fixture / "stderr").read_text()
    success = not expected_stdout and not expected_stderr
    result = run([str(linter), f"./{fixture.name}"], work, success=success)
    stdout = normalized(result.stdout, work)
    stderr = normalized(result.stderr, work)
    assert stdout == expected_stdout, stdout
    assert stderr == expected_stderr, stderr


def assert_nil_fact_cache_sequence(linter, work):
    """Keep imported nil facts after an unrelated TGo package analysis."""
    assert_case(linter, work, FIXTURE / "modernizegood")
    assert_case(linter, work, FIXTURE / "nilbad")


def write_invalid_consumers(work):
    sources = {
        "invaliddirect": 'package invaliddirect\nimport _ "example.com/tgolint/model"\n',
        "invalidmiddle": 'package invalidmiddle\nimport _ "example.com/tgolint/model"\n',
        "invalidtransitive": (
            'package invalidtransitive\n'
            'import _ "example.com/tgolint/invalidmiddle"\n'
        ),
    }
    for name, source in sources.items():
        directory = work / name
        directory.mkdir()
        (directory / f"{name.removeprefix('invalid')}.go").write_text(source)


def diagnostics(linter, work, package):
    result = run([str(linter), package], work, success=False)
    return normalized(result.stdout + result.stderr, work)


def assert_invalid_consumers(linter, work):
    direct = diagnostics(linter, work, "./invaliddirect")
    assert (
        "dependency example.com/tgolint/model failed tgo verification" in direct
    ), direct
    transitive = diagnostics(linter, work, "./invalidtransitive")
    assert (
        "dependency example.com/tgolint/invalidmiddle failed tgo verification"
        in transitive
    ), transitive


def assert_integrity_checks(linter, work):
    ordinary = work / "nestedmodel" / "model_tgo.go"
    ordinary_text = ordinary.read_text()
    ordinary.write_text(ordinary_text.replace("Name string", "Name int", 1))
    result = diagnostics(linter, work, "./nestedmodel")
    assert "generated tgo output integrity check failed" in result
    ordinary.write_text(ordinary_text)

    generated_model = work / "model" / "model_tgo.go"
    model_source = work / "model" / "model.tgo"
    generated_text = generated_model.read_text()

    replacements = (
        ("return Count{value: value}.check()", "return Count{}.check()"),
        (
            "return Event{tgoTag: EventTagStarted, tgoStarted: value}",
            "_ = value\n\treturn Event{}",
        ),
        (
            "func Identity(value int) int { return value }",
            "func Identity(value int) int { return 0 }",
        ),
    )
    for original, replacement in replacements:
        forged = generated_text.replace(original, replacement, 1)
        assert forged != generated_text
        generated_model.write_text(forged)
        result = diagnostics(linter, work, "./model")
        assert "generated tgo output integrity check failed" in result
        generated_model.write_text(generated_text)

    missing_source = model_source.with_suffix(".tgo.missing")
    model_source.rename(missing_source)
    assert_invalid_consumers(linter, work)
    missing_source.rename(model_source)

    model_source_text = model_source.read_text()
    model_source.write_text(model_source_text.replace("value.value <= 0", "value.value <= 10", 1))
    result = diagnostics(linter, work, "./model")
    assert "generated tgo output integrity check failed" in result
    model_source.write_text(model_source_text)

    whitespace = model_source_text.replace("package model\n\n", "package model\n\n\n", 1)
    model_source.write_text(whitespace)
    run([str(linter), "./model"], work)
    model_source.write_text(model_source_text)

    stale = model_source_text.replace(
        "type Count struct { value int } checked",
        "type Count struct { value int; extra int } checked",
    )
    stale = stale.replace("Started struct", "Opened struct")
    stale = stale.replace('json:"pair"', 'json:"stale"')
    model_source.write_text(stale)
    result = diagnostics(linter, work, "./model")
    assert "generated tgo output integrity check failed" in result
    assert_invalid_consumers(linter, work)


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
        write_invalid_consumers(work)

        for fixture in fixture_directories():
            assert_case(linter, work, fixture)

        assert_nil_fact_cache_sequence(linter, work)

        assert_integrity_checks(linter, work)


if __name__ == "__main__":
    main()
