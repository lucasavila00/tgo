from __future__ import annotations

import subprocess
import tempfile
import unittest
from pathlib import Path

from scripts import check_tgofmt


class ProductionSourcesTest(unittest.TestCase):
    def test_discovers_source_outside_old_roots_and_excludes_fixtures(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            repository = Path(temporary)
            paths = [
                "new-package/model.tgo",
                "new-package/testdata/input.tgo",
                "third_party/go/model.tgo",
            ]
            for relative in paths:
                path = repository / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("package model\n", encoding="utf-8")

            sources = check_tgofmt.production_sources(repository)

            self.assertEqual(set(sources), {Path("new-package/model.tgo")})


class StagedSourcesTest(unittest.TestCase):
    def test_reads_staged_source_that_is_absent_from_the_worktree(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            repository = Path(temporary)
            subprocess.run(["git", "init", "-q"], cwd=repository, check=True)
            source = repository / "package" / "model.tgo"
            source.parent.mkdir()
            source.write_text("package model\n", encoding="utf-8")
            subprocess.run(["git", "add", "."], cwd=repository, check=True)
            subprocess.run(
                [
                    "git",
                    "-c",
                    "user.name=TGo Test",
                    "-c",
                    "user.email=test@example.com",
                    "commit",
                    "-q",
                    "-m",
                    "Initial source",
                ],
                cwd=repository,
                check=True,
            )
            source.write_text("package staged\n", encoding="utf-8")
            subprocess.run(["git", "add", "."], cwd=repository, check=True)
            source.unlink()

            sources = check_tgofmt.staged_sources(repository)

            self.assertEqual(sources, {Path("package/model.tgo"): b"package staged\n"})

    def test_excludes_indexed_fixture_and_nested_module_source(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            repository = Path(temporary)
            subprocess.run(["git", "init", "-q"], cwd=repository, check=True)
            paths = {
                "package/model.tgo": "package model\n",
                "package/testdata/input.tgo": "fixture\n",
                "nested/go.mod": "module nested\n",
                "nested/model.tgo": "package nested\n",
            }
            for relative, content in paths.items():
                path = repository / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(content, encoding="utf-8")
            subprocess.run(["git", "add", "."], cwd=repository, check=True)

            sources = check_tgofmt.staged_sources(repository)

            self.assertEqual(sources, {Path("package/model.tgo"): b"package model\n"})


class NoncanonicalSourcesTest(unittest.TestCase):
    def test_reports_source_that_tgofmt_changes(self) -> None:
        sources = {
            Path("new-package/model.tgo"): b"package model\n\nfunc  value( )  { }\n",
        }

        failures = check_tgofmt.noncanonical_sources(sources)

        self.assertEqual(failures, [Path("new-package/model.tgo")])

    def test_accepts_canonical_source(self) -> None:
        sources = {Path("new-package/model.tgo"): b"package model\n"}

        failures = check_tgofmt.noncanonical_sources(sources)

        self.assertEqual(failures, [])

    def test_diagnostic_names_the_file_and_correction_command(self) -> None:
        message = check_tgofmt.failure_message(Path("new-package/model.tgo"))

        self.assertEqual(
            message,
            "new-package/model.tgo: run "
            "'go run ./cmd/tgofmt -w -- new-package/model.tgo'",
        )


if __name__ == "__main__":
    unittest.main()
