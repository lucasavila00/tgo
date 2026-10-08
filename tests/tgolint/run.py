#!/usr/bin/env python3
"""Build tgo and tgolint, then check Go boundary diagnostics."""

from pathlib import Path
import hashlib
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


def with_generated_digest(text, source):
    parts = text.split("\n", 3)
    digest = hashlib.sha256(
        source + b"\x00tgo generated body\x00" + parts[3].encode()
    ).hexdigest()
    parts[1] = f'//tgo:v1 "model.tgo" {digest}'
    return "\n".join(parts)


def write_invalid_consumers(work):
    direct = work / "invaliddirect"
    middle = work / "invalidmiddle"
    transitive = work / "invalidtransitive"
    direct.mkdir()
    middle.mkdir()
    transitive.mkdir()
    (direct / "direct.go").write_text(
        'package invaliddirect\nimport _ "example.com/tgolint/model"\n'
    )
    (middle / "middle.go").write_text(
        'package invalidmiddle\nimport _ "example.com/tgolint/model"\n'
    )
    (transitive / "transitive.go").write_text(
        'package invalidtransitive\nimport _ "example.com/tgolint/invalidmiddle"\n'
    )


def assert_invalid_consumers(linter, work):
    direct = run([str(linter), "./invaliddirect"], work, success=False)
    direct_diagnostics = normalized(direct.stdout + direct.stderr, work)
    assert (
        "dependency example.com/tgolint/model failed tgo verification"
        in direct_diagnostics
    ), direct_diagnostics
    transitive = run([str(linter), "./invalidtransitive"], work, success=False)
    transitive_diagnostics = normalized(transitive.stdout + transitive.stderr, work)
    assert (
        "dependency example.com/tgolint/invalidmiddle failed tgo verification"
        in transitive_diagnostics
    ), transitive_diagnostics


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

        generated_model = work / "model" / "model_tgo.go"
        model_source = work / "model" / "model.tgo"
        generated_text = generated_model.read_text()
        for original, replacement in (
            ("return Count{value: value}, nil", "return Count{}, nil"),
            (
                "return Event{tgoTag: 1, tgoStarted: value}",
                "_ = value\n\treturn Event{}",
            ),
        ):
            forged = generated_text.replace(original, replacement, 1)
            assert forged != generated_text
            forged = with_generated_digest(forged, model_source.read_bytes())
            generated_model.write_text(forged)
            forged_result = run([str(linter), "./model"], work, success=False)
            forged_diagnostics = normalized(
                forged_result.stdout + forged_result.stderr, work
            )
            assert "does not match the current compiler emitter" in forged_diagnostics
            generated_model.write_text(generated_text)

        metadata_line = next(
            line for line in generated_text.splitlines() if line.startswith("//tgo:v1 ")
        )
        missing_metadata = generated_text.replace(metadata_line + "\n", "", 1)
        assert missing_metadata != generated_text
        generated_model.write_text(missing_metadata)
        missing_result = run([str(linter), "./model"], work, success=False)
        missing_diagnostics = normalized(
            missing_result.stdout + missing_result.stderr, work
        )
        assert "generated tgo model metadata is invalid" in missing_diagnostics
        assert_invalid_consumers(linter, work)
        generated_model.write_text(generated_text)

        corrupt_metadata = generated_text.replace('"model.tgo"', "bad", 1)
        assert corrupt_metadata != generated_text
        generated_model.write_text(corrupt_metadata)
        corrupt_result = run([str(linter), "./model"], work, success=False)
        corrupt_diagnostics = normalized(
            corrupt_result.stdout + corrupt_result.stderr, work
        )
        assert "generated tgo model metadata is invalid" in corrupt_diagnostics
        generated_model.write_text(generated_text)

        missing_source = model_source.with_suffix(".tgo.missing")
        model_source.rename(missing_source)
        assert_invalid_consumers(linter, work)
        missing_source.rename(model_source)

        model_source_text = model_source.read_text()
        model_source.write_text(model_source_text.replace("value > 0", "value > 10", 1))
        generated_model.write_text(
            with_generated_digest(generated_text, model_source.read_bytes())
        )
        predicate_result = run([str(linter), "./model"], work, success=False)
        predicate_diagnostics = normalized(
            predicate_result.stdout + predicate_result.stderr, work
        )
        assert "does not match the current compiler emitter" in predicate_diagnostics
        model_source.write_text(model_source_text)
        generated_model.write_text(generated_text)

        stale = model_source.read_text()
        stale = stale.replace(
            "type Count int where value > 0",
            'type Count string where value != ""',
        )
        stale = stale.replace("Started struct", "Opened struct")
        stale = stale.replace('json:"pair"', 'json:"stale"')
        model_source.write_text(stale)
        generated_model.write_text(
            with_generated_digest(generated_text, model_source.read_bytes())
        )
        mismatch = run([str(linter), "./model"], work, success=False)
        diagnostics = normalized(mismatch.stdout + mismatch.stderr, work)
        assert "does not match the current compiler emitter" in diagnostics
        assert_invalid_consumers(linter, work)


if __name__ == "__main__":
    main()
