#!/usr/bin/env python3
"""Test the repository dead-code policy."""

from __future__ import annotations

import unittest
from pathlib import Path

import check_dead_code


ROOT = Path(__file__).resolve().parents[1]
FIXTURE = ROOT / "scripts" / "testdata" / "deadcode"


class DeadCodeTest(unittest.TestCase):
    """Check source mapping, live variants, and exclusions."""

    @classmethod
    def setUpClass(cls) -> None:
        cls.records = check_dead_code.deadcode(["./scripts/testdata/deadcode"])

    def names(self) -> set[str]:
        return {
            str(function["Name"])
            for package in self.records
            for function in package["Funcs"]
        }

    def test_policy(self) -> None:
        exclusions = {
            ("scripts/testdata/deadcode/main.go", "PublicAPI"):
                "approved public fixture API",
        }
        diagnostics = check_dead_code.check(self.records, ROOT, exclusions)
        self.assertEqual(
            diagnostics,
            [
                "scripts/testdata/deadcode/main.go:15: unreachable func: "
                "deadGo (Go source)",
                "scripts/testdata/deadcode/model.tgo:3: unreachable func: "
                "fixtureOwner.deadGeneratedSupport (TGo source; generated at "
                "scripts/testdata/deadcode/model_tgo.go:9)",
                "scripts/testdata/deadcode/model.tgo:5: unreachable func: "
                "deadTGo (TGo source; generated at "
                "scripts/testdata/deadcode/model_tgo.go:7)",
            ],
        )

    def test_fixture_has_required_variants(self) -> None:
        self.assertTrue((FIXTURE / "main_test.go").is_file())
        self.assertTrue((FIXTURE / "platform_linux.go").is_file())
        self.assertTrue((FIXTURE / "platform_other.go").is_file())
        self.assertNotIn("testOnly", self.names())
        self.assertNotIn("platformOnly", self.names())
        self.assertIn("fixtureOwner.MarshalJSON", self.names())


if __name__ == "__main__":
    unittest.main()
