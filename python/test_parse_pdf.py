#!/usr/bin/env python3
"""Unit tests for parse_pdf.py financial extraction functions."""

import unittest
from unittest.mock import patch, MagicMock
from parse_pdf import (
    find_section,
    find_rpt_section,
    extract_pnl,
    extract_balance_sheet,
    extract_cash_flow,
    extract_highlights,
    extract_financials,
    completeness_score,
    missing_fields,
    extract_financials_hybrid,
    COMPLETENESS_THRESHOLD,
)


SAMPLE_PNL_TEXT = """
STATEMENT OF STANDALONE PROFIT AND LOSS
For the year ended 31 March 2024

Revenue from Operations                     1,23,456
Other Income                                   2,500
Total Income                               1,25,956
Cost of materials consumed                  45,000
Employee benefit expenses                   18,000
Finance costs                                3,200
Depreciation and amortisation expense        4,800
Other expenses                              22,000
Total expenses                              93,000
Profit before exceptional items and tax     32,956
Tax expense                                  8,240
Profit for the year                         24,716
"""

SAMPLE_BS_TEXT = """
STANDALONE BALANCE SHEET
As at 31 March 2024

Assets
Property, plant and equipment              80,000
Total current assets                       55,000
Cash and cash equivalents                  12,500
Inventories                                15,000
Trade receivables                          18,000
Total assets                              1,50,000

Equity and Liabilities
Total equity                               70,000
Long-term borrowings                       30,000
Total current liabilities                  50,000
"""

SAMPLE_CF_TEXT = """
STANDALONE STATEMENT OF CASH FLOWS
For the year ended 31 March 2024

Net cash generated from operating activities   28,000
Net cash used in investing activities         -15,000
Net cash used in financing activities          -8,000
Net increase in cash and cash equivalents       5,000
"""

SAMPLE_HIGHLIGHTS_TEXT = """
KEY FINANCIAL HIGHLIGHTS

Earnings per share (Basic)          45.20
Book value per share               320.50
Dividend per share                  12.00
Return on equity                    18.50
Return on capital employed          22.30
Debt-to-equity ratio                 0.43
"""

SAMPLE_RPT_TEXT = """
Some preamble text...

RELATED PARTY TRANSACTIONS

Transactions with Acme Private Limited (subsidiary):
Purchase of goods: Rs. 1,200 Cr

Transactions with Beta Holdings Limited (holding company):
Dividend paid: Rs. 500 Cr

NOTES TO ACCOUNTS
"""

COMBINED_TEXT = (
    SAMPLE_PNL_TEXT
    + "\n\n"
    + SAMPLE_BS_TEXT
    + "\n\n"
    + SAMPLE_CF_TEXT
    + "\n\n"
    + SAMPLE_HIGHLIGHTS_TEXT
    + "\n\n"
    + SAMPLE_RPT_TEXT
)


class TestFindSection(unittest.TestCase):
    def test_finds_pnl_section(self):
        section = find_section(COMBINED_TEXT, r"statement\s+of\s+standalone\s+profit\s+and\s+loss")
        self.assertIn("Revenue from Operations", section)
        self.assertIn("1,23,456", section)

    def test_finds_balance_sheet_section(self):
        section = find_section(COMBINED_TEXT, r"standalone\s+balance\s+sheet")
        self.assertIn("Total assets", section)

    def test_finds_cash_flow_section(self):
        section = find_section(COMBINED_TEXT, r"cash\s+flow")
        self.assertIn("operating activities", section)

    def test_returns_empty_string_when_not_found(self):
        section = find_section("Some random text without headers.", r"profit\s+and\s+loss")
        self.assertEqual(section, "")

    def test_finds_highlights_section(self):
        section = find_section(COMBINED_TEXT, r"key\s+financial\s+highlights")
        self.assertIn("Earnings per share", section)


class TestFindRptSection(unittest.TestCase):
    def test_finds_rpt_section(self):
        section = find_rpt_section(SAMPLE_RPT_TEXT)
        self.assertIn("Acme Private Limited", section)

    def test_returns_full_text_when_no_rpt_header(self):
        text = "No related party info here."
        section = find_rpt_section(text)
        self.assertEqual(section, text)


class TestExtractPnl(unittest.TestCase):
    def test_extracts_revenue_from_operations(self):
        result = extract_pnl(SAMPLE_PNL_TEXT)
        self.assertIn("revenueFromOperations", result)
        self.assertEqual(result["revenueFromOperations"], "123456")

    def test_extracts_profit_after_tax(self):
        result = extract_pnl(SAMPLE_PNL_TEXT)
        self.assertIn("profitAfterTax", result)
        self.assertEqual(result["profitAfterTax"], "24716")

    def test_extracts_total_expenses(self):
        result = extract_pnl(SAMPLE_PNL_TEXT)
        self.assertIn("totalExpenses", result)
        self.assertEqual(result["totalExpenses"], "93000")

    def test_extracts_tax_expense(self):
        result = extract_pnl(SAMPLE_PNL_TEXT)
        self.assertIn("taxExpense", result)
        self.assertEqual(result["taxExpense"], "8240")

    def test_extracts_employee_benefits(self):
        result = extract_pnl(SAMPLE_PNL_TEXT)
        self.assertIn("employeeBenefits", result)
        self.assertEqual(result["employeeBenefits"], "18000")

    def test_missing_section_returns_empty_dict(self):
        result = extract_pnl("This text has no profit and loss section.")
        self.assertEqual(result, {})


class TestExtractBalanceSheet(unittest.TestCase):
    def test_extracts_total_assets(self):
        result = extract_balance_sheet(SAMPLE_BS_TEXT)
        self.assertIn("totalAssets", result)
        self.assertEqual(result["totalAssets"], "150000")

    def test_extracts_cash(self):
        result = extract_balance_sheet(SAMPLE_BS_TEXT)
        self.assertIn("cash", result)
        self.assertEqual(result["cash"], "12500")

    def test_extracts_current_assets(self):
        result = extract_balance_sheet(SAMPLE_BS_TEXT)
        self.assertIn("currentAssets", result)
        self.assertEqual(result["currentAssets"], "55000")

    def test_extracts_total_equity(self):
        result = extract_balance_sheet(SAMPLE_BS_TEXT)
        self.assertIn("totalEquity", result)
        self.assertEqual(result["totalEquity"], "70000")

    def test_extracts_long_term_debt(self):
        result = extract_balance_sheet(SAMPLE_BS_TEXT)
        self.assertIn("longTermDebt", result)
        self.assertEqual(result["longTermDebt"], "30000")

    def test_missing_section_returns_empty_dict(self):
        result = extract_balance_sheet("No balance sheet here.")
        self.assertEqual(result, {})


class TestExtractCashFlow(unittest.TestCase):
    def test_extracts_from_operations(self):
        result = extract_cash_flow(SAMPLE_CF_TEXT)
        self.assertIn("fromOperations", result)
        self.assertEqual(result["fromOperations"], "28000")

    def test_extracts_net_change(self):
        result = extract_cash_flow(SAMPLE_CF_TEXT)
        self.assertIn("netChange", result)
        self.assertEqual(result["netChange"], "5000")

    def test_missing_section_returns_empty_dict(self):
        result = extract_cash_flow("No cash flow section here.")
        self.assertEqual(result, {})


class TestExtractHighlights(unittest.TestCase):
    def test_extracts_eps(self):
        result = extract_highlights(SAMPLE_HIGHLIGHTS_TEXT)
        self.assertIn("eps", result)
        self.assertEqual(result["eps"], "45.20")

    def test_extracts_roe(self):
        result = extract_highlights(SAMPLE_HIGHLIGHTS_TEXT)
        self.assertIn("roe", result)
        self.assertEqual(result["roe"], "18.50")

    def test_extracts_roce(self):
        result = extract_highlights(SAMPLE_HIGHLIGHTS_TEXT)
        self.assertIn("roce", result)
        self.assertEqual(result["roce"], "22.30")

    def test_extracts_book_value_per_share(self):
        result = extract_highlights(SAMPLE_HIGHLIGHTS_TEXT)
        self.assertIn("bookValuePerShare", result)
        self.assertEqual(result["bookValuePerShare"], "320.50")

    def test_extracts_dividend_per_share(self):
        result = extract_highlights(SAMPLE_HIGHLIGHTS_TEXT)
        self.assertIn("dividendPerShare", result)
        self.assertEqual(result["dividendPerShare"], "12.00")

    def test_extracts_debt_to_equity(self):
        result = extract_highlights(SAMPLE_HIGHLIGHTS_TEXT)
        self.assertIn("debtToEquity", result)
        self.assertEqual(result["debtToEquity"], "0.43")


class TestExtractFinancials(unittest.TestCase):
    def test_returns_all_sections(self):
        result = extract_financials(COMBINED_TEXT)
        self.assertIn("pnl", result)
        self.assertIn("balanceSheet", result)
        self.assertIn("cashFlow", result)
        self.assertIn("highlights", result)

    def test_empty_sections_are_dicts_not_errors(self):
        result = extract_financials("Random text with no financial data.")
        self.assertIsInstance(result["pnl"], dict)
        self.assertIsInstance(result["balanceSheet"], dict)
        self.assertIsInstance(result["cashFlow"], dict)
        self.assertIsInstance(result["highlights"], dict)

    def test_missing_section_returns_empty_dict_not_none(self):
        result = extract_financials("Random text with no financial data.")
        # Sections that aren't found return empty dicts, not None or errors
        for key in ("pnl", "balanceSheet", "cashFlow", "highlights"):
            self.assertEqual(result[key], {}, f"{key} should be empty dict, not {result[key]!r}")


class TestCompletenessScore(unittest.TestCase):

    def test_empty_financials_scores_zero(self):
        f = {"pnl": {}, "balanceSheet": {}, "cashFlow": {}, "highlights": {}}
        self.assertEqual(completeness_score(f), 0)

    def test_partial_financials_scores_correctly(self):
        f = {
            "pnl": {"revenueFromOperations": "100", "profitAfterTax": "20"},
            "balanceSheet": {"totalAssets": "500"},
            "cashFlow": {},
            "highlights": {},
        }
        self.assertEqual(completeness_score(f), 3)

    def test_full_financials_scores_above_threshold(self):
        f = {
            "pnl": {
                "revenueFromOperations": "1", "otherIncome": "2", "totalIncome": "3",
                "materialCost": "4", "employeeBenefits": "5", "financeCosts": "6",
                "depreciation": "7", "otherExpenses": "8", "totalExpenses": "9",
                "profitBeforeTax": "10", "taxExpense": "11", "profitAfterTax": "12",
            },
            "balanceSheet": {
                "totalAssets": "1", "fixedAssets": "2", "currentAssets": "3",
                "cash": "4", "inventory": "5", "receivables": "6",
                "totalEquity": "7", "longTermDebt": "8", "currentLiabilities": "9",
            },
            "cashFlow": {
                "fromOperations": "1", "fromInvesting": "2",
                "fromFinancing": "3", "netChange": "4",
            },
            "highlights": {
                "eps": "1", "bookValuePerShare": "2", "dividendPerShare": "3",
                "roe": "4", "roce": "5", "debtToEquity": "6",
            },
        }
        self.assertGreaterEqual(completeness_score(f), COMPLETENESS_THRESHOLD)


class TestMissingFields(unittest.TestCase):

    def test_all_missing_when_empty(self):
        f = {"pnl": {}, "balanceSheet": {}, "cashFlow": {}, "highlights": {}}
        gaps = missing_fields(f)
        self.assertIn("pnl", gaps)
        self.assertIn("revenueFromOperations", gaps["pnl"])

    def test_no_missing_when_fully_populated(self):
        f = {
            "pnl": {k: "1" for k in ["revenueFromOperations", "otherIncome", "totalIncome",
                                      "materialCost", "employeeBenefits", "financeCosts",
                                      "depreciation", "otherExpenses", "totalExpenses",
                                      "profitBeforeTax", "taxExpense", "profitAfterTax"]},
            "balanceSheet": {k: "1" for k in ["totalAssets", "fixedAssets", "currentAssets",
                                               "cash", "inventory", "receivables",
                                               "totalEquity", "longTermDebt", "currentLiabilities"]},
            "cashFlow": {k: "1" for k in ["fromOperations", "fromInvesting", "fromFinancing", "netChange"]},
            "highlights": {k: "1" for k in ["eps", "bookValuePerShare", "dividendPerShare",
                                             "roe", "roce", "debtToEquity"]},
        }
        self.assertEqual(missing_fields(f), {})

    def test_partially_populated_section(self):
        f = {
            "pnl": {"revenueFromOperations": "100"},
            "balanceSheet": {}, "cashFlow": {}, "highlights": {},
        }
        gaps = missing_fields(f)
        self.assertNotIn("revenueFromOperations", gaps.get("pnl", []))
        self.assertIn("profitAfterTax", gaps.get("pnl", []))


class TestHybridExtractor(unittest.TestCase):

    def test_no_claude_when_disabled(self):
        """With use_claude=False, result has no _claudeFilled entries even when sparse."""
        result = extract_financials_hybrid("No financial data here.", use_claude=False)
        self.assertEqual(result["_claudeFilled"], [])

    def test_no_claude_when_score_above_threshold(self):
        """Claude is not called when regex already fills enough fields."""
        # Build text that populates >= COMPLETENESS_THRESHOLD fields via regex
        rich_text = "\n".join([
            "STATEMENT OF STANDALONE PROFIT AND LOSS",
            "Revenue from Operations 100000",
            "Other Income 5000",
            "Total Income 105000",
            "Cost of materials consumed 40000",
            "Employee benefit expenses 15000",
            "Finance costs 3000",
            "Depreciation and amortisation expense 4000",
            "Other expenses 10000",
            "Total expenses 72000",
            "Profit before tax 33000",
            "Tax expense 8000",
            "Profit for the year 25000",
        ])
        called = []

        def fake_claude(*args, **kwargs):
            called.append(True)
            return {}, []

        with patch("parse_pdf.claude_fill_gaps", side_effect=fake_claude):
            extract_financials_hybrid(rich_text, use_claude=True, claude_api_key="fake-key")

        self.assertEqual(called, [], "Claude should not be called when regex score is sufficient")

    def test_claude_called_when_sparse(self):
        """Claude is called when regex extracts fewer than COMPLETENESS_THRESHOLD fields."""
        called = []

        def fake_claude(full_text, gaps, api_key):
            called.append(True)
            # Simulate Claude returning a P&L field.
            return {"pnl": {"revenueFromOperations": "999"}}, ["pnl.revenueFromOperations"]

        with patch("parse_pdf.claude_fill_gaps", side_effect=fake_claude):
            result = extract_financials_hybrid(
                "Sparse text with no financial data.",
                use_claude=True,
                claude_api_key="fake-key",
            )

        self.assertEqual(called, [True], "Claude should be called when regex score is below threshold")
        self.assertEqual(result["pnl"].get("revenueFromOperations"), "999")
        self.assertIn("pnl.revenueFromOperations", result["_claudeFilled"])

    def test_regex_values_not_overwritten_by_claude(self):
        """When regex already has a value, Claude's value for the same field is ignored."""
        rich_text = "\n".join([
            "STATEMENT OF STANDALONE PROFIT AND LOSS",
            "Revenue from Operations 12345",
        ])

        def fake_claude(full_text, gaps, api_key):
            # Claude tries to set a different value for revenueFromOperations.
            return {"pnl": {"revenueFromOperations": "99999"}}, ["pnl.revenueFromOperations"]

        # Force sparse score so Claude is called by patching completeness_score.
        with patch("parse_pdf.completeness_score", return_value=0), \
             patch("parse_pdf.claude_fill_gaps", side_effect=fake_claude):
            result = extract_financials_hybrid(rich_text, use_claude=True, claude_api_key="fake-key")

        # Regex value "12345" must survive; Claude's "99999" must be discarded.
        self.assertEqual(result["pnl"].get("revenueFromOperations"), "12345")

    def test_claude_filled_list_tracks_fields(self):
        """_claudeFilled correctly lists all fields Claude populated."""
        def fake_claude(full_text, gaps, api_key):
            return {
                "pnl": {"profitAfterTax": "5000"},
                "cashFlow": {"fromOperations": "8000"},
            }, ["pnl.profitAfterTax", "cashFlow.fromOperations"]

        with patch("parse_pdf.claude_fill_gaps", side_effect=fake_claude):
            result = extract_financials_hybrid(
                "Sparse.",
                use_claude=True,
                claude_api_key="fake-key",
            )

        self.assertIn("pnl.profitAfterTax", result["_claudeFilled"])
        self.assertIn("cashFlow.fromOperations", result["_claudeFilled"])


if __name__ == "__main__":
    unittest.main()
