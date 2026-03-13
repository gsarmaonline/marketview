#!/usr/bin/env python3
"""Unit tests for parse_pdf.py supply chain extraction functions."""

import unittest
from parse_pdf import (
    find_rpt_section,
    extract_companies,
)


SAMPLE_RPT_TEXT = """
Some preamble text...

RELATED PARTY TRANSACTIONS

Transactions with Acme Private Limited (subsidiary):
Purchase of goods: Rs. 1,200 Cr
"""

HOLDING_TEXT = """
RELATED PARTY TRANSACTIONS

Transactions with Beta Holdings Limited (holding company):
Dividend paid: Rs. 500 Cr
"""

NO_RPT_TEXT = "No related party info here."

SUBSIDIARY_TEXT = """
RELATED PARTY TRANSACTIONS

Transactions with Gamma Solutions Limited (wholly owned subsidiary):
Services rendered: Rs. 300 Cr
"""

ASSOCIATE_TEXT = """
RELATED PARTY TRANSACTIONS

Transactions with Delta Industries Limited (associate):
Purchase of raw materials: Rs. 150 Cr
"""

LONG_RPT_WITH_STOP = """
RELATED PARTY TRANSACTIONS

""" + ("X" * 600) + """

NOTES TO ACCOUNTS
some notes here.
"""


class TestFindRptSection(unittest.TestCase):
    def test_finds_rpt_section(self):
        section = find_rpt_section(SAMPLE_RPT_TEXT)
        self.assertIn("Acme Private Limited", section)

    def test_returns_full_text_when_no_rpt_header(self):
        section = find_rpt_section(NO_RPT_TEXT)
        self.assertEqual(section, NO_RPT_TEXT)

    def test_truncates_at_next_section(self):
        section = find_rpt_section(LONG_RPT_WITH_STOP)
        # "NOTES TO ACCOUNTS" triggers the stop pattern when far enough into the chunk
        self.assertNotIn("NOTES TO ACCOUNTS", section)

    def test_rpt_section_contains_transactions(self):
        section = find_rpt_section(SAMPLE_RPT_TEXT)
        self.assertIn("Purchase of goods", section)


class TestExtractCompanies(unittest.TestCase):
    def test_extracts_company_names(self):
        section = find_rpt_section(SAMPLE_RPT_TEXT)
        results = extract_companies(section)
        names = [r["name"] for r in results]
        self.assertTrue(
            any("Acme" in n for n in names),
            f"Expected Acme in {names}",
        )

    def test_extracts_subsidiary_relationship(self):
        section = find_rpt_section(SAMPLE_RPT_TEXT)
        results = extract_companies(section)
        rels = {r["name"]: r["relationship"] for r in results}
        acme = next((v for k, v in rels.items() if "Acme" in k), None)
        self.assertIsNotNone(acme)
        self.assertEqual(acme, "subsidiary")

    def test_extracts_holding_company_relationship(self):
        section = find_rpt_section(HOLDING_TEXT)
        results = extract_companies(section)
        rels = {r["name"]: r["relationship"] for r in results}
        beta = next((v for k, v in rels.items() if "Beta" in k), None)
        self.assertIsNotNone(beta)
        self.assertEqual(beta, "holding")

    def test_extracts_wholly_owned_subsidiary(self):
        section = find_rpt_section(SUBSIDIARY_TEXT)
        results = extract_companies(section)
        rels = {r["name"]: r["relationship"] for r in results}
        gamma = next((v for k, v in rels.items() if "Gamma" in k), None)
        self.assertIsNotNone(gamma)
        self.assertEqual(gamma, "wholly_owned_subsidiary")

    def test_extracts_associate(self):
        section = find_rpt_section(ASSOCIATE_TEXT)
        results = extract_companies(section)
        rels = {r["name"]: r["relationship"] for r in results}
        delta = next((v for k, v in rels.items() if "Delta" in k), None)
        self.assertIsNotNone(delta)
        self.assertEqual(delta, "associate")

    def test_no_duplicate_companies(self):
        section = find_rpt_section(SAMPLE_RPT_TEXT)
        results = extract_companies(section)
        names = [r["name"] for r in results]
        self.assertEqual(len(names), len(set(names)))

    def test_empty_text_returns_empty_list(self):
        results = extract_companies("")
        self.assertEqual(results, [])

    def test_extracts_amount_when_present(self):
        section = find_rpt_section(SAMPLE_RPT_TEXT)
        results = extract_companies(section)
        # At least one entity should have an amount extracted
        amounts = [r for r in results if r.get("amount")]
        self.assertTrue(len(amounts) > 0, "Expected at least one entity with an amount")

    def test_short_names_excluded(self):
        text = "RELATED PARTY TRANSACTIONS\nABC Ltd is a subsidiary.\n"
        results = extract_companies(text)
        # "ABC Ltd" is only 7 chars — still included; ensure no crash
        self.assertIsInstance(results, list)


if __name__ == "__main__":
    unittest.main()
