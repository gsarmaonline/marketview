#!/usr/bin/env python3
"""parse_pdf.py - Extract Related Party Transactions from Indian annual report PDFs.

Usage:  python3 parse_pdf.py <pdf_url_or_path>
Output: JSON to stdout:
    {"companies": [{"name": "...", "relationship": "...", "amount": "..."}, ...]}
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
        print(json.dumps({"companies": companies}))

    except Exception as exc:
        print(json.dumps({"error": str(exc)}))
        sys.exit(1)

    finally:
        if is_temp and pdf_path and os.path.exists(pdf_path):
            os.unlink(pdf_path)


if __name__ == "__main__":
    main()
