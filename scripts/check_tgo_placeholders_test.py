from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from scripts import check_tgo_placeholders


class TGoPlaceholderTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.repository = Path(self.temporary.name)

    def tearDown(self) -> None:
        self.temporary.cleanup()

    def write(self, relative: str, source: str) -> None:
        path = self.repository / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(source, encoding="utf-8")

    def test_rejects_numbered_placeholder_in_production_tgo(self) -> None:
        self.write(
            "pkg/model.tgo",
            "package model\nvar enumValue42 = 1\n",
        )

        result = check_tgo_placeholders.failures(self.repository)

        self.assertEqual(
            result,
            [
                "pkg/model.tgo:2: replace numbered enum placeholder "
                "with a role name"
            ],
        )

    def test_accepts_role_name(self) -> None:
        self.write("internal/model/model.tgo", "package model\nvar nodeValue = 1\n")

        self.assertEqual(check_tgo_placeholders.failures(self.repository), [])

    def test_ignores_test_sources_and_generated_go(self) -> None:
        source = "package model\nvar enumValue42 = 1\n"
        for relative in (
            "tests/example.tgo",
            "pkg/testdata/example.tgo",
            "pkg/fixtures/example.tgo",
            "pkg/model_test.tgo",
            "pkg/model_tgo.go",
        ):
            self.write(relative, source)

        self.assertEqual(check_tgo_placeholders.failures(self.repository), [])


if __name__ == "__main__":
    unittest.main()
