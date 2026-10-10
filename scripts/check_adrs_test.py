import unittest

from scripts import check_adrs


class CheckContentTest(unittest.TestCase):
    def test_accepts_small_proposal_with_issue(self) -> None:
        failures = check_adrs.check_content(
            "docs/adr/example.md",
            "# Decision\n\nIssue: #135\n\n## Decision\n\nUse one API.\n",
            True,
        )

        self.assertEqual(failures, [])

    def test_rejects_long_prose_but_allows_long_code(self) -> None:
        failures = check_adrs.check_content(
            "docs/adr/example.md",
            "x" * 81 + "\n\n```text\n" + "x" * 81 + "\n```\n\nIssue: #135\n",
            True,
        )

        self.assertEqual(len(failures), 1)
        self.assertIn("prose limit is 80", failures[0])

    def test_rejects_more_than_one_hundred_lines(self) -> None:
        content = "Issue: #135\n" + "line\n" * 100

        failures = check_adrs.check_content(
            "docs/adr/example.md", content, True
        )

        self.assertIn("101 lines; ADR maximum is 100", failures[0])

    def test_rejects_template_placeholder(self) -> None:
        failures = check_adrs.check_content(
            "docs/adr/example.md", "Issue: #135\n<Decision title>\n", True
        )

        self.assertIn("remove template instruction placeholders", failures[0])

    def test_rejects_issue_number_placeholder(self) -> None:
        failures = check_adrs.check_content(
            "docs/adr/example.md", "Issue: #135\n<number>\n", True
        )

        self.assertIn("remove template instruction placeholders", failures[0])

    def test_accepts_other_angle_bracket_forms(self) -> None:
        failures = check_adrs.check_content(
            "docs/adr/example.md",
            "Issue: #135\n<details>\nUse <T>.\n</details>\n",
            True,
        )

        self.assertEqual(failures, [])

    def test_rejects_status_field(self) -> None:
        failures = check_adrs.check_content(
            "docs/adr/example.md", "Issue: #135\nStatus: accepted\n", True
        )

        self.assertIn("remove the Status field", failures[0])

    def test_rejects_missing_issue_reference(self) -> None:
        failures = check_adrs.check_content(
            "docs/adr/example.md", "# Decision\n", True
        )

        self.assertIn("add an issue reference", failures[0])

    def test_checks_permanent_files_without_proposal_fields(self) -> None:
        failures = check_adrs.check_content(
            "docs/adr/template.md", "# <Decision title>\n", False
        )

        self.assertEqual(failures, [])


if __name__ == "__main__":
    unittest.main()
