#!/usr/bin/env python3
"""parse_pdf.py - Extract Related Party Transactions and financials from Indian annual report PDFs.

Usage:  python3 parse_pdf.py <pdf_url_or_path>
Output: JSON to stdout:
    {"companies": [...], "financials": {...}}
    {"error": "..."}   on failure

Dependencies:
    pip install pdfplumber pdf2image pytesseract
    System: poppler-utils (for pdf2image), tesseract-ocr (for OCR fallback)
"""

import sys
import json
import re
import os
import tempfile
import urllib.request


# ---------------------------------------------------------------------------
# PDF download
# ---------------------------------------------------------------------------

def download_pdf(url: str) -> str:
    req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
    with tempfile.NamedTemporaryFile(suffix=".pdf", delete=False) as f:
        with urllib.request.urlopen(req, timeout=60) as resp:
            f.write(resp.read())
        return f.name


# ---------------------------------------------------------------------------
# Text extraction
# ---------------------------------------------------------------------------

def _extract_pdfplumber(pdf_path: str) -> str:
    import pdfplumber
    pages = []
    with pdfplumber.open(pdf_path) as pdf:
        for page in pdf.pages:
            t = page.extract_text()
            if t:
                pages.append(t)
    return "\n".join(pages)


def _extract_ocr(pdf_path: str) -> str:
    from pdf2image import convert_from_path
    import pytesseract
    images = convert_from_path(pdf_path, dpi=200)
    return "\n".join(pytesseract.image_to_string(img) for img in images)


def extract_text(pdf_path: str) -> str:
    """Try pdfplumber; fall back to OCR when text looks too sparse (scanned PDF)."""
    text = _extract_pdfplumber(pdf_path)
    page_count = max(1, text.count("\f") + 1)
    if len(text) / page_count < 200:
        try:
            text = _extract_ocr(pdf_path)
        except Exception:
            pass  # keep pdfplumber result
    return text


# ---------------------------------------------------------------------------
# RPT section isolation
# ---------------------------------------------------------------------------

_RPT_HEADER = re.compile(
    r"related\s+party\s+(?:transactions?|disclosures?)"
    r"|transactions?\s+with\s+related\s+part(?:y|ies)",
    re.IGNORECASE,
)

# Heuristic: next major section starts with an all-caps heading or numbered note
_NEXT_SECTION = re.compile(r"\n(?:[A-Z][A-Z\s]{15,}|(?:Note|NOTE)\s+\d+)\b")


def find_rpt_section(text: str) -> str:
    m = _RPT_HEADER.search(text)
    if not m:
        return text  # search full text as fallback
    chunk = text[m.start() : m.start() + 20_000]
    stop = _NEXT_SECTION.search(chunk, 500)
    return chunk[: stop.start()] if stop else chunk


# ---------------------------------------------------------------------------
# Generic section finder
# ---------------------------------------------------------------------------

# Signals that a section match is an actual financial statement, not a TOC
# entry, footnote reference, or ancillary table.
_STATEMENT_SIGNALS = re.compile(
    r"for\s+the\s+year\s+ended"
    r"|as\s+at\s+\d"
    r"|(?:₹|Rs?\.?|C)\s*in\s+(?:crore|lakh)"
    r"|\(\s*(?:₹|Rs?\.?|C)\s*in\s+crore"
    r"|notes?\s+20\d\d"          # "Notes 2024-25" style column header
    r"|2024.25\s+2023.24"        # year-over-year column header
    r"|revenue\s+from\s+operations"
    r"|total\s+assets",
    re.IGNORECASE,
)


def find_section(text: str, header_pattern: str, max_chars: int = 15_000) -> str:
    """Return the text starting from the best match of header_pattern up to the
    next major section.

    Scores each occurrence by how many financial-statement signals appear in the
    first 600 characters after the header. This skips Table-of-Contents hits and
    ancillary tables (which can have high digit density but lack statement signals)
    and lands on the actual statement page.
    """
    best_chunk = ""
    best_score = -1
    for m in re.finditer(header_pattern, text, re.IGNORECASE):
        chunk = text[m.start() : m.start() + max_chars]
        stop = _NEXT_SECTION.search(chunk, 200)
        chunk = chunk[: stop.start()] if stop else chunk
        score = len(_STATEMENT_SIGNALS.findall(chunk[:600]))
        if score > best_score:
            best_score = score
            best_chunk = chunk
    return best_chunk


# ---------------------------------------------------------------------------
# Company name & relationship extraction
# ---------------------------------------------------------------------------

_SUFFIXES = (
    r"Private\s+Limited|Pvt\.?\s+Ltd\.?|Limited|Ltd\.?"
    r"|LLP|LLC|Inc\.?|Corp(?:oration)?\.?"
    r"|Enterprises|Industries|Holdings|Ventures"
    r"|Solutions|Services|Technologies|Infra(?:structure)?"
)

_COMPANY_RE = re.compile(
    r"\b([A-Z][A-Za-z&.\-]*(?:\s+[A-Z][A-Za-z&.\-]*){0,6}"
    r"\s+(?:" + _SUFFIXES + r"))\b"
)

_RELATIONSHIP_HINTS: list[tuple[str, str]] = [
    (r"wholly.owned\s+subsidiary", "wholly_owned_subsidiary"),
    (r"subsidiary", "subsidiary"),
    (r"associate", "associate"),
    (r"joint\s+venture", "joint_venture"),
    (r"holding\s+company|parent", "holding"),
    (r"key\s+management", "key_management"),
    (r"director", "key_management"),
    (r"promoter", "promoter"),
    (r"supplier|vendor", "supplier"),
    (r"customer|purchaser|client", "customer"),
]

_AMOUNT_RE = re.compile(
    r"(?:₹|Rs\.?\s*)?(\d[\d,]*(?:\.\d+)?)\s*(?:Cr(?:ore)?s?|Lakh(?:s)?)?",
    re.IGNORECASE,
)


def _classify(context: str) -> str:
    ctx = context.lower()
    for pattern, label in _RELATIONSHIP_HINTS:
        if re.search(pattern, ctx):
            return label
    return "related_party"


def _amount(context: str) -> str:
    m = _AMOUNT_RE.search(context)
    return m.group(0).strip() if m else ""


def extract_companies(rpt_text: str) -> list[dict]:
    seen: set[str] = set()
    results: list[dict] = []
    for m in _COMPANY_RE.finditer(rpt_text):
        name = re.sub(r"\s+", " ", m.group(1)).strip()
        if name in seen or len(name) < 5:
            continue
        seen.add(name)
        s = max(0, m.start() - 300)
        e = min(len(rpt_text), m.end() + 300)
        ctx = rpt_text[s:e]
        entry: dict = {"name": name, "relationship": _classify(ctx)}
        amt = _amount(ctx)
        if amt:
            entry["amount"] = amt
        results.append(entry)
    return results


# ---------------------------------------------------------------------------
# Financial data extraction helpers
# ---------------------------------------------------------------------------

# Pattern to capture a numeric value (with optional commas/decimals) following a label.
# Matches patterns like:  "Revenue from Operations  1,23,456.78"
#                         "Revenue from Operations : 1,23,456"
def _find_value(text: str, label_pattern: str) -> str:
    """Find the numeric value that appears after label_pattern.

    Indian annual reports place a note-reference number between the label and
    the value, e.g.:
        Revenue from operations   12   2,55,324
    The optional group ``(?:\\d{1,3}\\s+)?`` skips a 1-3 digit note reference
    (followed by whitespace) so the actual amount is captured instead.
    """
    pattern = (
        r"(?:" + label_pattern + r")"
        r"[^\n\d]{0,60}?"          # non-digit gap (label punctuation / spaces)
        r"(?:\d{1,3}\s+)?"         # optional note-reference number (e.g. "12 ")
        r"([\d,]+(?:\.\d+)?)"      # the actual value
    )
    m = re.search(pattern, text, re.IGNORECASE)
    if m and m.group(1) is not None:
        return m.group(1).replace(",", "")
    return ""


# ---------------------------------------------------------------------------
# P&L extraction
# ---------------------------------------------------------------------------

_PNL_HEADERS = re.compile(
    r"statement\s+of\s+(?:standalone\s+)?profit\s+(?:and|&)\s+loss"
    r"|profit\s+(?:and|&)\s+loss\s+(?:account|statement)"
    r"|income\s+statement",
    re.IGNORECASE,
)

_PNL_LABELS: dict[str, str] = {
    "revenueFromOperations": r"revenue\s+from\s+operations",
    "otherIncome":           r"other\s+income",
    "totalIncome":           r"total\s+(?:income|revenue)",
    "materialCost":          r"(?:cost\s+of\s+(?:material|goods)|material\s+(?:cost|consumed)|raw\s+material)",
    "employeeBenefits":      r"employee\s+(?:benefit|cost|expense)",
    "financeCosts":          r"finance\s+(?:cost|charge)",
    "depreciation":          r"depreciation\s+(?:and\s+amortisation|&\s+amortisation|expense)?",
    "otherExpenses":         r"other\s+expenses",
    "totalExpenses":         r"total\s+expenses",
    "profitBeforeTax":       r"profit\s+before\s+(?:exceptional.*?and\s+)?tax",
    "taxExpense":            r"tax\s+expense",
    "profitAfterTax":        r"profit\s+(?:for\s+the\s+(?:year|period)|after\s+tax)",
}


def extract_pnl(text: str) -> dict:
    section = find_section(text, _PNL_HEADERS.pattern, max_chars=10_000)
    if not section:
        return {}
    result: dict = {}
    for key, pattern in _PNL_LABELS.items():
        val = _find_value(section, pattern)
        if val:
            result[key] = val
    return result


# ---------------------------------------------------------------------------
# Balance Sheet extraction
# ---------------------------------------------------------------------------

_BS_HEADERS = re.compile(
    r"(?:standalone\s+)?balance\s+sheet"
    r"|statement\s+of\s+(?:standalone\s+)?financial\s+position",
    re.IGNORECASE,
)

_BS_LABELS: dict[str, str] = {
    "totalAssets":        r"total\s+assets",
    "fixedAssets":        r"(?:property,\s+plant|fixed\s+assets|tangible\s+assets)",
    "currentAssets":      r"total\s+current\s+assets",
    "cash":               r"cash\s+(?:and\s+cash\s+equivalents?|&\s+cash\s+equivalents?)",
    "inventory":          r"inventor(?:y|ies)",
    "receivables":        r"trade\s+receivables?",
    "totalEquity":        r"total\s+(?:equity|shareholder|shareholders)[^\n\d]{0,40}",
    "longTermDebt":       r"(?:long.term\s+borrowings?|non.current\s+borrowings?)",
    "currentLiabilities": r"total\s+current\s+liabilities",
}


def extract_balance_sheet(text: str) -> dict:
    section = find_section(text, _BS_HEADERS.pattern, max_chars=10_000)
    if not section:
        return {}
    result: dict = {}
    for key, pattern in _BS_LABELS.items():
        val = _find_value(section, pattern)
        if val:
            result[key] = val
    return result


# ---------------------------------------------------------------------------
# Cash Flow extraction
# ---------------------------------------------------------------------------

_CF_HEADERS = re.compile(
    r"(?:standalone\s+)?(?:statement\s+of\s+)?cash\s+flow",
    re.IGNORECASE,
)

_CF_LABELS: dict[str, str] = {
    "fromOperations": r"(?:net\s+)?cash\s+(?:generated\s+)?from\s+operating",
    "fromInvesting":  r"(?:net\s+)?cash\s+(?:used\s+in|from)\s+investing",
    "fromFinancing":  r"(?:net\s+)?cash\s+(?:used\s+in|from)\s+financing",
    "netChange":      r"net\s+(?:increase|decrease)\s+in\s+cash",
}


def extract_cash_flow(text: str) -> dict:
    section = find_section(text, _CF_HEADERS.pattern, max_chars=8_000)
    if not section:
        return {}
    result: dict = {}
    for key, pattern in _CF_LABELS.items():
        val = _find_value(section, pattern)
        if val:
            result[key] = val
    return result


# ---------------------------------------------------------------------------
# Financial Highlights extraction
# ---------------------------------------------------------------------------

_HIGHLIGHTS_HEADERS = re.compile(
    r"(?:key\s+)?financial\s+(?:highlights?|ratios?|summary)"
    r"|per\s+share\s+data",
    re.IGNORECASE,
)

_HIGHLIGHTS_LABELS: dict[str, str] = {
    "eps":               r"(?:basic\s+)?(?:earning|earnings)\s+per\s+share|EPS",
    "bookValuePerShare": r"book\s+value\s+per\s+share",
    "dividendPerShare":  r"dividend\s+per\s+share",
    "roe":               r"return\s+on\s+(?:net\s+worth|equity)|ROE",
    "roce":              r"return\s+on\s+capital\s+employed|ROCE",
    "debtToEquity":      r"debt.to.equity(?:\s+ratio)?|D/E\s+ratio",
}


def extract_highlights(text: str) -> dict:
    section = find_section(text, _HIGHLIGHTS_HEADERS.pattern, max_chars=5_000)
    if not section:
        section = text  # highlights may not have a dedicated header; search full text
    result: dict = {}
    for key, pattern in _HIGHLIGHTS_LABELS.items():
        val = _find_value(section, pattern)
        if val:
            result[key] = val
    return result


# ---------------------------------------------------------------------------
# Master financials extractor
# ---------------------------------------------------------------------------

def extract_financials(text: str) -> dict:
    """Extract all financial data from the full PDF text using regex only."""
    pnl = extract_pnl(text)
    bs = extract_balance_sheet(text)
    cf = extract_cash_flow(text)
    highlights = extract_highlights(text)
    return {
        "pnl": pnl,
        "balanceSheet": bs,
        "cashFlow": cf,
        "highlights": highlights,
    }


# ---------------------------------------------------------------------------
# Completeness scoring
# ---------------------------------------------------------------------------

# All extractable fields, grouped by section key.
_ALL_FIELDS: dict[str, list[str]] = {
    "pnl":          list(_PNL_LABELS.keys()),
    "balanceSheet": list(_BS_LABELS.keys()),
    "cashFlow":     list(_CF_LABELS.keys()),
    "highlights":   list(_HIGHLIGHTS_LABELS.keys()),
}

# Trigger Claude when fewer than this many fields are populated.
COMPLETENESS_THRESHOLD = 8


def completeness_score(financials: dict) -> int:
    """Count how many fields across all sections have a non-empty value."""
    count = 0
    for section, fields in _ALL_FIELDS.items():
        section_data = financials.get(section, {})
        count += sum(1 for f in fields if section_data.get(f))
    return count


def missing_fields(financials: dict) -> dict[str, list[str]]:
    """Return {section: [field, ...]} for every field that is empty."""
    missing: dict[str, list[str]] = {}
    for section, fields in _ALL_FIELDS.items():
        section_data = financials.get(section, {})
        gaps = [f for f in fields if not section_data.get(f)]
        if gaps:
            missing[section] = gaps
    return missing


# ---------------------------------------------------------------------------
# Claude gap-fill
# ---------------------------------------------------------------------------

# Section text keys sent to Claude (pre-extracted to keep token count down).
_SECTION_HEADER_PATTERNS: dict[str, str] = {
    "pnl":          _PNL_HEADERS.pattern,
    "balanceSheet": _BS_HEADERS.pattern,
    "cashFlow":     _CF_HEADERS.pattern,
    "highlights":   _HIGHLIGHTS_HEADERS.pattern,
}
_SECTION_MAX_CHARS = 8_000


def _extract_section_text(full_text: str, section: str) -> str:
    pattern = _SECTION_HEADER_PATTERNS.get(section, "")
    if not pattern:
        return ""
    # Re-use find_section so both paths benefit from the TOC-skip logic.
    return find_section(full_text, pattern, _SECTION_MAX_CHARS)


def claude_fill_gaps(
    full_text: str,
    gaps: dict[str, list[str]],
    api_key: str,
) -> tuple[dict, list[str]]:
    """
    Call Claude to extract only the fields that regex missed.

    Returns:
        (filled_values, claude_filled_keys)
        filled_values     — {section: {field: value}} for fields Claude found
        claude_filled_keys — flat list of "section.field" strings that Claude populated
    """
    import anthropic

    # Build a compact representation of what we need, with the relevant text.
    sections_payload = []
    for section, fields in gaps.items():
        section_text = _extract_section_text(full_text, section)
        if not section_text:
            continue
        sections_payload.append({
            "section": section,
            "fields":  fields,
            "text":    section_text,
        })

    if not sections_payload:
        return {}, []

    prompt = (
        "You are extracting financial data from an Indian company annual report.\n"
        "For each section below, extract only the listed fields from the provided text.\n"
        "Return a JSON object with this exact shape:\n"
        '{"pnl": {}, "balanceSheet": {}, "cashFlow": {}, "highlights": {}}\n'
        "Rules:\n"
        "- Values must be plain numbers only (no units, no commas, no currency symbols).\n"
        "- Use the standalone/unconsolidated figure when both are present.\n"
        "- If a field cannot be found, omit it (do not return null or empty string).\n"
        "- Do not add any fields beyond those listed.\n\n"
    )
    for sp in sections_payload:
        prompt += f"=== {sp['section']} — extract: {sp['fields']} ===\n{sp['text']}\n\n"

    client = anthropic.Anthropic(api_key=api_key)
    message = client.messages.create(
        model="claude-haiku-4-5-20251001",
        max_tokens=1024,
        messages=[{"role": "user", "content": prompt}],
    )

    raw = message.content[0].text.strip()
    # Strip markdown fences if present.
    if raw.startswith("```"):
        raw = re.sub(r"^```[a-z]*\n?", "", raw)
        raw = re.sub(r"\n?```$", "", raw.strip())

    try:
        filled = json.loads(raw)
    except json.JSONDecodeError:
        return {}, []

    # Validate: only keep fields that were actually requested.
    claude_filled_keys: list[str] = []
    clean: dict[str, dict] = {}
    for section, fields in gaps.items():
        section_data = filled.get(section, {})
        for field in fields:
            val = section_data.get(field)
            if val is not None and str(val).strip():
                clean.setdefault(section, {})[field] = str(val)
                claude_filled_keys.append(f"{section}.{field}")

    return clean, claude_filled_keys


# ---------------------------------------------------------------------------
# Hybrid extractor (regex first, Claude for gaps)
# ---------------------------------------------------------------------------

def extract_financials_hybrid(
    full_text: str,
    use_claude: bool = False,
    claude_api_key: str = "",
) -> dict:
    """
    Run regex extraction first. If the result is sparse and Claude is enabled,
    call Claude to fill in the missing fields.

    The returned dict includes a '_claudeFilled' key listing every field that
    Claude populated — useful for tracking which regex patterns still need work.
    """
    result = extract_financials(full_text)
    result["_claudeFilled"] = []

    if not use_claude or not claude_api_key:
        return result

    score = completeness_score(result)
    if score >= COMPLETENESS_THRESHOLD:
        return result  # regex was good enough

    gaps = missing_fields(result)
    if not gaps:
        return result

    filled, claude_filled_keys = claude_fill_gaps(full_text, gaps, claude_api_key)

    # Merge: regex values take priority; Claude fills empty slots only.
    for section, values in filled.items():
        for field, value in values.items():
            if not result.get(section, {}).get(field):
                result.setdefault(section, {})[field] = value

    result["_claudeFilled"] = claude_filled_keys
    return result


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def main() -> None:
    if len(sys.argv) < 2:
        print(json.dumps({"error": "usage: parse_pdf.py <pdf_url_or_path>"}))
        sys.exit(1)

    source = sys.argv[1]
    pdf_path: str | None = None
    is_temp = False

    try:
        if source.startswith("http"):
            pdf_path = download_pdf(source)
            is_temp = True
        else:
            pdf_path = source

        text = extract_text(pdf_path)
        rpt = find_rpt_section(text)
        companies = extract_companies(rpt)
        financials = extract_financials(text)
        print(json.dumps({"companies": companies, "financials": financials}))

    except Exception as exc:
        print(json.dumps({"error": str(exc)}))
        sys.exit(1)

    finally:
        if is_temp and pdf_path and os.path.exists(pdf_path):
            os.unlink(pdf_path)


if __name__ == "__main__":
    main()
