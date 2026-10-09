#!/usr/bin/env python3
"""Test the removed-syntax documentation check."""

import unittest

import check_removed_syntax


class RemovedSyntaxTest(unittest.TestCase):
    """Check rejected claims and valid uses of the identifier."""

    def test_rejects_active_claims(self) -> None:
        files = {
            "docs/spec/README.md": (
                "`where` is contextual after a checked base type.\n"
                "type Port int where value > 0\n"
            ),
            "editors/vscode/syntaxes/tgo.tmLanguage.json": (
                '"name": "keyword.control.where.tgo"\n'
            ),
        }
        self.assertEqual(len(check_removed_syntax.violations(files)), 3)

    def test_allows_prose_and_identifiers(self) -> None:
        files = {
            "docs/guide/README.md": "Use the package where the value belongs.\n",
            "editors/vscode/syntaxes/tgo.tmLanguage.json": "{}\n",
            "fixture.txt": "type WhereAlias where\nwhere := 1\n",
        }
        self.assertEqual(check_removed_syntax.violations(files), [])


if __name__ == "__main__":
    unittest.main()
