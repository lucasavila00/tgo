from __future__ import annotations

import subprocess
import tempfile
import unittest
from pathlib import Path

from scripts import formatter_ci


class ClassificationTest(unittest.TestCase):
    def test_each_declared_file_requires_the_corpus(self) -> None:
        for path in sorted(formatter_ci.DEPENDENCY_FILES):
            with self.subTest(path=path):
                self.assertTrue(formatter_ci.affects_formatter([path]))

    def test_each_declared_directory_requires_the_corpus(self) -> None:
        for directory in formatter_ci.DEPENDENCY_DIRECTORIES:
            path = directory + "change.go"
            with self.subTest(path=path):
                self.assertTrue(formatter_ci.affects_formatter([path]))

    def test_unrelated_documentation_does_not_require_the_corpus(self) -> None:
        self.assertFalse(formatter_ci.affects_formatter(["docs/guide/README.md"]))

    def test_similar_prefix_does_not_require_the_corpus(self) -> None:
        self.assertFalse(formatter_ci.affects_formatter(["pkg/syntax-old/model.go"]))

    def test_any_declared_path_requires_the_corpus(self) -> None:
        paths = ["docs/guide/README.md", "pkg/format/printer.tgo"]

        self.assertTrue(formatter_ci.affects_formatter(paths))

    def test_changed_paths_uses_the_pull_request_merge_base(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            repository = Path(temporary)
            self.git(repository, "init", "-q")
            (repository / "README.md").write_text("initial\n")
            self.commit(repository, "Initial source")
            initial = self.git(repository, "rev-parse", "HEAD")
            self.git(repository, "switch", "-q", "-c", "feature", initial)
            self.git(repository, "switch", "-q", "-")
            formatter = repository / "pkg" / "format" / "printer.tgo"
            formatter.parent.mkdir(parents=True)
            formatter.write_text("package format\n")
            self.commit(repository, "Base formatter change")
            base = self.git(repository, "rev-parse", "HEAD")
            self.git(repository, "switch", "-q", "feature")
            docs = repository / "docs" / "README.md"
            docs.parent.mkdir()
            docs.write_text("docs\n")
            self.commit(repository, "Feature documentation")
            head = self.git(repository, "rev-parse", "HEAD")

            paths = formatter_ci.changed_paths(repository, base, head)

            self.assertEqual(paths, ["docs/README.md"])

    @staticmethod
    def git(repository: Path, *arguments: str) -> str:
        return subprocess.check_output(
            ["git", *arguments],
            cwd=repository,
            text=True,
        ).strip()

    @classmethod
    def commit(cls, repository: Path, message: str) -> None:
        cls.git(repository, "add", ".")
        cls.git(
            repository,
            "-c",
            "user.name=TGo Test",
            "-c",
            "user.email=test@example.com",
            "commit",
            "-q",
            "-m",
            message,
        )


class GateTest(unittest.TestCase):
    def test_accepts_successful_required_corpus(self) -> None:
        failure = formatter_ci.gate_failure("success", "true", "success")

        self.assertIsNone(failure)

    def test_accepts_skipped_unneeded_corpus(self) -> None:
        failure = formatter_ci.gate_failure("success", "false", "skipped")

        self.assertIsNone(failure)

    def test_rejects_skipped_required_corpus(self) -> None:
        failure = formatter_ci.gate_failure("success", "true", "skipped")

        self.assertEqual(failure, "required formatter corpus ended with skipped")

    def test_rejects_failed_required_corpus(self) -> None:
        failure = formatter_ci.gate_failure("success", "true", "failure")

        self.assertEqual(failure, "required formatter corpus ended with failure")

    def test_rejects_unneeded_corpus_that_ran(self) -> None:
        failure = formatter_ci.gate_failure("success", "false", "success")

        self.assertEqual(failure, "unneeded formatter corpus ended with success")

    def test_rejects_failed_classification(self) -> None:
        failure = formatter_ci.gate_failure("failure", "", "skipped")

        self.assertEqual(
            failure,
            "formatter change classification ended with failure",
        )

    def test_rejects_missing_classification_output(self) -> None:
        failure = formatter_ci.gate_failure("success", "", "skipped")

        self.assertEqual(
            failure,
            "formatter change classification returned ''",
        )


if __name__ == "__main__":
    unittest.main()
