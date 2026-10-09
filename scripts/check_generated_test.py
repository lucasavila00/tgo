from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from scripts import check_generated


class ProductionFilesTest(unittest.TestCase):
    def test_discovers_source_outside_old_roots(self) -> None:
        repository = (
            Path(__file__).parent
            / "testdata"
            / "check-generated"
            / "repository"
        )

        files = check_generated.production_files(repository, ".tgo")

        self.assertEqual(set(files), {Path("new-package/model.tgo")})

    def test_excludes_non_production_trees(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            repository = Path(temporary)
            paths = [
                "package/model.tgo",
                "package/testdata/model.tgo",
                "third_party/go/model.tgo",
                "bin/model.tgo",
                ".cache/model.tgo",
                "_temporary/model.tgo",
            ]
            for relative in paths:
                path = repository / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(relative, encoding="utf-8")

            files = check_generated.production_files(repository, ".tgo")

            self.assertEqual(set(files), {Path("package/model.tgo")})

    def test_excludes_nested_modules(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            repository = Path(temporary)
            source = repository / "nested" / "model.tgo"
            source.parent.mkdir()
            source.write_text("package model\n", encoding="utf-8")
            (source.parent / "go.mod").write_text("module nested\n", encoding="utf-8")

            files = check_generated.production_files(repository, ".tgo")

            self.assertEqual(files, {})

    def test_classifies_production_paths_without_the_worktree(self) -> None:
        nested_modules = {Path("nested")}

        self.assertTrue(
            check_generated.production_path(Path("new-package/model.tgo"), nested_modules)
        )
        self.assertFalse(
            check_generated.production_path(
                Path("new-package/testdata/model.tgo"),
                nested_modules,
            )
        )
        self.assertFalse(
            check_generated.production_path(Path("nested/model.tgo"), nested_modules)
        )


class RequireEqualTest(unittest.TestCase):
    def test_reports_missing_output(self) -> None:
        with self.assertRaisesRegex(SystemExit, "missing generated output: model_tgo.go"):
            check_generated.require_equal({}, {Path("model_tgo.go"): b"generated"})

    def test_reports_stale_or_edited_output(self) -> None:
        with self.assertRaisesRegex(
            SystemExit,
            "stale or edited generated output: model_tgo.go",
        ):
            check_generated.require_equal(
                {Path("model_tgo.go"): b"old"},
                {Path("model_tgo.go"): b"new"},
            )

    def test_reports_orphan_output(self) -> None:
        with self.assertRaisesRegex(SystemExit, "orphan generated output: model_tgo.go"):
            check_generated.require_equal({Path("model_tgo.go"): b"generated"}, {})


if __name__ == "__main__":
    unittest.main()
