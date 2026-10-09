from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from scripts import check_source_size


class SourceSizeTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.repository = Path(self.temporary.name)

    def tearDown(self) -> None:
        self.temporary.cleanup()

    def write_lines(self, relative: str, count: int) -> Path:
        path = self.repository / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text("line\n" * count, encoding="utf-8")
        return path

    def test_rejects_oversized_handwritten_go_and_tgo(self) -> None:
        self.write_lines("package/model.go", check_source_size.MAX_LINES + 1)
        self.write_lines("package/model.tgo", check_source_size.MAX_LINES + 1)

        result = check_source_size.failures(self.repository, {})

        self.assertEqual(len(result), 2)
        self.assertIn("package/model.go", result[0])
        self.assertIn("package/model.tgo", result[1])

    def test_ignores_generated_go(self) -> None:
        self.write_lines("package/model_tgo.go", check_source_size.MAX_LINES + 1)

        self.assertEqual(check_source_size.failures(self.repository, {}), [])

    def test_requires_an_explicit_fixture_exclusion(self) -> None:
        relative = Path("package/testdata/large.go")
        self.write_lines(str(relative), check_source_size.MAX_LINES + 1)

        result = check_source_size.failures(self.repository, {})

        self.assertEqual(len(result), 1)
        self.assertIn(str(relative), result[0])

    def test_accepts_a_documented_fixture_exclusion(self) -> None:
        relative = Path("package/testdata/large.go")
        self.write_lines(str(relative), check_source_size.MAX_LINES + 1)

        result = check_source_size.failures(
            self.repository,
            {relative: "The parser stress fixture must have more than 700 lines."},
        )

        self.assertEqual(result, [])

    def test_rejects_an_exclusion_without_a_reason(self) -> None:
        relative = Path("package/testdata/large.go")
        self.write_lines(str(relative), check_source_size.MAX_LINES + 1)

        result = check_source_size.failures(self.repository, {relative: ""})

        self.assertTrue(any("needs a reason" in failure for failure in result))
        self.assertTrue(any("maximum is" in failure for failure in result))

    def test_rejects_a_stale_exclusion(self) -> None:
        relative = Path("package/testdata/small.go")
        self.write_lines(str(relative), check_source_size.MAX_LINES)

        result = check_source_size.failures(
            self.repository,
            {relative: "This reason is now stale."},
        )

        self.assertEqual(len(result), 1)
        self.assertIn("stale fixture exclusion", result[0])


if __name__ == "__main__":
    unittest.main()
