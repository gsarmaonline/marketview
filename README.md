# marketview

A tool for Indian market investors to assess whether now is a good time to invest, based on live market indicators, with a manual portfolio tracker.

## Features

- Market indicators (NIFTY 50 PE ratio, signal scoring)
- Live market news feed (Economic Times, Moneycontrol, Business Standard) with per-stock news pipeline
- Portfolio management: stocks, FDs, mutual funds, gold, and other assets
- Mutual fund deep research: holdings breakdown, NAV history, allocation stats
- Per-stock deep research: annual reports (NSE with BSE fallback) with supply chain extraction from Related Party Transactions

## Architecture

- **Go backend** (`main.go`, `internal/`) - fetches and scores market indicators, manages portfolio data, exposes a JSON API on `:8080`
  - `internal/api` - HTTP server (Gin), route registration, CORS middleware
  - `internal/indicators` - market indicator framework (NIFTY 50 PE)
  - `internal/mutualfund` - mutual fund search and holdings (mfapi.in + Yahoo Finance)
  - `internal/news` - RSS news aggregator (Economic Times, Moneycontrol, Business Standard) + in-memory stock news pipeline (`Store`)
  - `internal/nse` - NSE India HTTP client
  - `internal/deepresearch` - per-stock deep research: annual reports via NSE (BSE fallback), PDF parsing, supply chain extraction
- **Python PDF parser** (`python/`) - long-running Flask HTTP service (`server.py`) on `:5001`; exposes `POST /parse` for supply chain extraction from annual report PDFs using `pdfplumber` with `pytesseract` OCR fallback for scanned PDFs
  - `internal/db` - PostgreSQL connection, startup migration (`schema.sql` embedded)
  - `internal/portfolio` - portfolio holdings CRUD
  - `internal/portfolio/db` - sqlc-generated type-safe query code (do not edit)
- **Next.js frontend** (`frontend/`) - indicators dashboard and portfolio management UI on `:3000`, with a live market news feed (tabbed: general market news and per-stock news) and stock deep research at `/stock/[symbol]`
- **PostgreSQL** - stores portfolio holdings

## Running

```bash
make up   # starts postgres, backend, and frontend via Docker Compose
```

Or locally:

```bash
# requires a local Postgres instance and the Python parser running
pip install -r python/requirements.txt   # first time only
python3 python/server.py                 # pdf-parser on :5001

export DB_HOST=localhost DB_USER=marketview DB_PASSWORD=marketview DB_NAME=marketview
go run main.go              # backend on :8080
cd frontend && npm run dev  # frontend on :3000
```

The backend reads `PARSER_URL` (default `http://localhost:5001`) to locate the PDF parser service.

## Development

### Regenerating database queries

SQL queries live in `db/queries/`. The schema lives in `internal/db/schema.sql`. After editing either, regenerate the Go code:

```bash
sqlc generate
```

The generated files in `internal/portfolio/db/` are committed and should not be edited by hand.

## API

### Market Indicators

`GET /api/indicators` — returns scored market indicators:

```json
[
  {
    "name": "NIFTY 50 PE Ratio",
    "value": 22.1,
    "unit": "x",
    "signal": "neutral",
    "description": "PE of 22.1x is in the fair-value range (20-25)"
  }
]
```

### News

`GET /api/news` — returns up to 20 recent market news items aggregated from Economic Times, Moneycontrol, and Business Standard.

`GET /api/news/stock/:symbol` — returns stock-specific news items stored in the in-memory pipeline for the given symbol (case-insensitive). Returns `[]` if no news has been ingested yet.

**Stock news pipeline:** Any ingestion source can push news into the store via `newsStore.Ingest(symbol, items)` or `newsStore.Replace(symbol, items)`. The store deduplicates by link and normalises symbols to uppercase.

### Mutual Funds

`GET /api/mutual-fund/search?q={name}` — search for funds by name

`GET /api/mutual-fund/{schemeCode}` — full fund details including stock holdings and allocation

Holdings and stats are sourced from Yahoo Finance and may be absent for funds not listed there.

### Portfolio

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/portfolio/holdings` | List all holdings |
| POST | `/api/portfolio/holdings` | Add a holding |
| PUT | `/api/portfolio/holdings/{id}` | Update a holding |
| DELETE | `/api/portfolio/holdings/{id}` | Delete a holding |

Supported asset types: `stock`, `fd`, `mutual_fund`, `gold`, `other`

Example payload:

```json
{
  "asset_type": "stock",
  "name": "RELIANCE",
  "quantity": 10,
  "buy_price": 2850.50,
  "current_value": 29500,
  "buy_date": "2024-01-15T00:00:00Z",
  "notes": "Long term hold",
  "metadata": {}
}
```

### Deep Research

`GET /api/stock/:symbol/deep-research` — annual reports for a stock (NSE symbol, e.g. `RELIANCE`). Falls back to BSE if NSE is unavailable:

```json
{
  "symbol": "RELIANCE",
  "annualReports": [
    {
      "seqNumber": 1,
      "issuer": "Reliance Industries Limited",
      "year": "2023-2024",
      "subject": "Annual Report 2023-24",
      "pdfLink": "https://archives.nseindia.com/..."
    }
  ],
  "annualReportsSource": "NSE",
  "parsedReportYear": "2023-2024",
  "supplyChain": [
    {
      "name": "Reliance Retail Ventures Limited",
      "relationship": "subsidiary"
    },
    {
      "name": "Jio Platforms Limited",
      "relationship": "subsidiary",
      "amount": "1234.56 Cr"
    }
  ]
}
```

`supplyChain` is populated by parsing the Related Party Transactions section of the most recent annual report PDF. Uses `python/parse_pdf.py` (requires `pip install -r python/requirements.txt`; for scanned PDFs also needs `tesseract` and `poppler` system packages).

## Indicators

| Indicator | Bullish | Neutral | Bearish |
|---|---|---|---|
| NIFTY 50 PE | < 20x | 20-25x | > 25x |
