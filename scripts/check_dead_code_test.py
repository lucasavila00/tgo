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
        packages = ["./scripts/testdata/deadcode"]
        cls.records = check_dead_code.deadcode(packages)
        cls.declarations = check_dead_code.dead_declarations(cls.records, packages)

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
            ("scripts/testdata/deadcode/main.go", "PublicType"):
                "approved public fixture type",
            ("scripts/testdata/deadcode/main.go", "PublicVar"):
                "approved public fixture variable",
            ("scripts/testdata/deadcode/main.go", "PublicConst"):
                "approved public fixture constant",
            ("scripts/testdata/deadcode/model_tgo.go", "fixtureOwnerTag"):
                "generated fixture protocol type",
            ("scripts/testdata/deadcode/model_tgo.go", "fixtureOwnerTagValue"):
                "generated fixture protocol constant",
        }
        diagnostics = check_dead_code.check(
            self.records, self.declarations, ROOT, exclusions
        )
        self.assertEqual(
            diagnostics,
            [
                "scripts/testdata/deadcode/main.go:15: unreachable func: "
                "deadGo (Go source)",
                "scripts/testdata/deadcode/main.go:17: unreachable type: "
                "deadGoType (Go source)",
                "scripts/testdata/deadcode/main.go:19: unreachable var: "
                "deadGoVar (Go source)",
                "scripts/testdata/deadcode/main.go:21: unreachable const: "
                "deadGoConst (Go source)",
                "scripts/testdata/deadcode/main.go:28: unreachable const: "
                "implicitSeed (Go source)",
                "scripts/testdata/deadcode/model.tgo:3: unreachable func: "
                "fixtureOwner.deadGeneratedSupport (TGo source; generated at "
                "scripts/testdata/deadcode/model_tgo.go:15)",
                "scripts/testdata/deadcode/model.tgo:3: unreachable type: "
                "fixtureOwner (TGo source; generated at "
                "scripts/testdata/deadcode/model_tgo.go:5)",
                "scripts/testdata/deadcode/model.tgo:5: unreachable func: "
                "deadTGo (TGo source; generated at "
                "scripts/testdata/deadcode/model_tgo.go:7)",
                "scripts/testdata/deadcode/model.tgo:7: unreachable type: "
                "firstOwner (TGo source; generated at "
                "scripts/testdata/deadcode/model_tgo.go:9)",
                "scripts/testdata/deadcode/model.tgo:8: unreachable type: "
                "secondOwner (TGo source; generated at "
                "scripts/testdata/deadcode/model_tgo.go:10)",
                "scripts/testdata/deadcode/model.tgo:10: unreachable func: "
                "firstOwner.sharedDead (TGo source; generated at "
                "scripts/testdata/deadcode/model_tgo.go:12)",
                "scripts/testdata/deadcode/model.tgo:11: unreachable func: "
                "secondOwner.sharedDead (TGo source; generated at "
                "scripts/testdata/deadcode/model_tgo.go:13)",
                "scripts/testdata/deadcode/model.tgo:13: unreachable type: "
                "deadTGoType (TGo source; generated at "
                "scripts/testdata/deadcode/model_tgo.go:19)",
                "scripts/testdata/deadcode/model.tgo:16: unreachable var: "
                "deadTGoVar (TGo source; generated at "
                "scripts/testdata/deadcode/model_tgo.go:22)",
                "scripts/testdata/deadcode/model.tgo:16: unreachable var: "
                "deadTGoVarSecond (TGo source; generated at "
                "scripts/testdata/deadcode/model_tgo.go:22)",
                "scripts/testdata/deadcode/model.tgo:20: unreachable const: "
                "deadTGoConst (TGo source; generated at "
                "scripts/testdata/deadcode/model_tgo.go:26)",
                "scripts/testdata/deadcode/model.tgo:20: unreachable const: "
                "deadTGoConstSecond (TGo source; generated at "
                "scripts/testdata/deadcode/model_tgo.go:26)",
            ],
        )

    def test_fixture_has_required_variants(self) -> None:
        self.assertTrue((FIXTURE / "main_test.go").is_file())
        self.assertTrue((FIXTURE / "platform_linux.go").is_file())
        self.assertTrue((FIXTURE / "platform_other.go").is_file())
        self.assertNotIn("testOnly", self.names())
        self.assertNotIn("platformOnly", self.names())
        self.assertIn("fixtureOwner.MarshalJSON", self.names())

    def test_exclusions_need_reasons_and_live_findings(self) -> None:
        with self.assertRaises(ValueError):
            check_dead_code.check([], [], ROOT, {("source.go", "Value"): ""})
        self.assertEqual(
            check_dead_code.check(
                [], [], ROOT, {("source.go", "Value"): "public fixture API"}
            ),
            ["stale dead-code exclusion: source.go: Value"],
        )


if __name__ == "__main__":
    unittest.main()
