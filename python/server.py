#!/usr/bin/env python3
"""server.py - HTTP server wrapping the PDF supply chain parser.

POST /parse
  Body:  {"url": "<pdf_url_or_local_path>"}
  Returns: {"companies": [...], "financials": {...}}  or  {"error": "..."}

Environment variables:
  PORT              — listening port (default 5001)
  ANTHROPIC_API_KEY — when set, enables Claude gap-fill for sparse regex results

Runs on 0.0.0.0:5001 by default; override with PORT env var.
"""

import os
from flask import Flask, request, jsonify
from parse_pdf import (
    download_pdf, extract_text, find_rpt_section,
    extract_companies, extract_financials_hybrid,
)

app = Flask(__name__)

_ANTHROPIC_API_KEY = os.environ.get("ANTHROPIC_API_KEY", "")
_USE_CLAUDE = bool(_ANTHROPIC_API_KEY)


@app.post("/parse")
def parse():
    body = request.get_json(silent=True) or {}
    url = body.get("url", "").strip()
    if not url:
        return jsonify({"error": "url is required"}), 400

    pdf_path = None
    is_temp = False
    try:
        if url.startswith("http"):
            pdf_path = download_pdf(url)
            is_temp = True
        else:
            pdf_path = url

        text = extract_text(pdf_path)
        rpt = find_rpt_section(text)
        companies = extract_companies(rpt)
        financials = extract_financials_hybrid(
            text,
            use_claude=_USE_CLAUDE,
            claude_api_key=_ANTHROPIC_API_KEY,
        )
        return jsonify({"companies": companies, "financials": financials})

    except Exception as exc:
        return jsonify({"error": str(exc)}), 500

    finally:
        if is_temp and pdf_path and os.path.exists(pdf_path):
            os.unlink(pdf_path)


@app.get("/health")
def health():
    return jsonify({"ok": True})


if __name__ == "__main__":
    port = int(os.environ.get("PORT", 5001))
    app.run(host="0.0.0.0", port=port)
