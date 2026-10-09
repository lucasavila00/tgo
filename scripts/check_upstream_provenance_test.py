from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from scripts import check_upstream_provenance


ADAPTATION = check_upstream_provenance.Adaptation(
    2016,
    "../../third_party/go/LICENSE",
    "docs/source-map.md",
    ("upstream.go",),
)
ADAPTATIONS = {"pkg/port/adapted.tgo": ADAPTATION}
SOURCE_MAP = "| `pkg/port/adapted.tgo` | `upstream.go` |\n"


class FailuresTest(unittest.TestCase):
    def repository(self, temporary: str, source: str, source_map: str) -> Path:
        repository = Path(temporary)
        adapted = repository / "pkg" / "port" / "adapted.tgo"
        adapted.parent.mkdir(parents=True)
        adapted.write_text(source, encoding="utf-8")
        documentation = repository / "docs" / "source-map.md"
        documentation.parent.mkdir()
        documentation.write_text(source_map, encoding="utf-8")
        return repository

    def test_accepts_complete_attribution(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            source = check_upstream_provenance.header(ADAPTATION) + "package port\n"
            repository = self.repository(temporary, source, SOURCE_MAP)

            self.assertEqual(
                check_upstream_provenance.failures(repository, ADAPTATIONS),
                [],
            )

    def test_rejects_missing_header(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            repository = self.repository(
                temporary,
                "package port\n",
                SOURCE_MAP,
            )

            self.assertIn(
                "pkg/port/adapted.tgo: adapted source has no required header",
                check_upstream_provenance.failures(repository, ADAPTATIONS),
            )

    def test_rejects_missing_source_map(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            source = check_upstream_provenance.header(ADAPTATION) + "package port\n"
            repository = self.repository(temporary, source, "# Source map\n")

            self.assertIn(
                "pkg/port/adapted.tgo: source map is absent from documentation",
                check_upstream_provenance.failures(repository, ADAPTATIONS),
            )

    def test_rejects_header_with_no_provenance_entry(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            source = check_upstream_provenance.header(ADAPTATION) + "package port\n"
            repository = self.repository(temporary, source, SOURCE_MAP)
            original = repository / "pkg" / "port" / "unrecorded.tgo"
            original.write_text(source, encoding="utf-8")

            self.assertIn(
                "pkg/port/unrecorded.tgo: Go Authors header has no provenance entry",
                check_upstream_provenance.failures(repository, ADAPTATIONS),
            )

    def test_accepts_original_source_without_header(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            source = check_upstream_provenance.header(ADAPTATION) + "package port\n"
            repository = self.repository(temporary, source, SOURCE_MAP)
            original = repository / "pkg" / "port" / "original.tgo"
            original.write_text("package port\n", encoding="utf-8")

            self.assertEqual(
                check_upstream_provenance.failures(repository, ADAPTATIONS),
                [],
            )

    def test_ignores_generated_outputs_but_checks_similar_go_names(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            source = check_upstream_provenance.header(ADAPTATION) + "package port\n"
            repository = self.repository(temporary, source, SOURCE_MAP)
            package = repository / "pkg" / "port"
            (package / "model_tgo_linux.go").write_text(source, encoding="utf-8")
            (package / "model_tgo_helper.go").write_text(source, encoding="utf-8")

            result = check_upstream_provenance.failures(repository, ADAPTATIONS)

            self.assertNotIn(
                "pkg/port/model_tgo_linux.go: Go Authors header has no "
                "provenance entry",
                result,
            )
            self.assertIn(
                "pkg/port/model_tgo_helper.go: Go Authors header has no "
                "provenance entry",
                result,
            )


if __name__ == "__main__":
    unittest.main()
