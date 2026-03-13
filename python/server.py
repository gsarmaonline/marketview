#!/usr/bin/env python3
"""server.py - HTTP server wrapping the PDF supply chain parser.

POST /parse
  Body:  {"url": "<pdf_url_or_local_path>"}
  Returns: {"companies": [...]}  or  {"error": "..."}

Environment variables:
  PORT  — listening port (default 5001)

Runs on 0.0.0.0:5001 by default; override with PORT env var.
"""

import os
from flask import Flask, request, jsonify
from parse_pdf import (
    download_pdf, extract_text, find_rpt_section,
    extract_companies,
)

app = Flask(__name__)


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
        return jsonify({"companies": companies})

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
