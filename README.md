# marketview

A tool for Indian market investors to assess whether now is a good time to invest, based on live market indicators, with a manual portfolio tracker.

## Features

- Market indicators (NIFTY 50 PE ratio, signal scoring)
- Live market news feed (Economic Times, Moneycontrol, Business Standard) with per-stock news pipeline
- Portfolio management: stocks, FDs, mutual funds, gold, and other assets
- Mutual fund deep research: holdings breakdown, NAV history, allocation stats
- Per-stock deep research: annual reports (NSE with BSE fallback), financial statements (P&L, Balance Sheet, Cash Flow, Highlights) fetched live from Yahoo Finance, supply chain extraction from Related Party Transactions via PDF parsing, and shareholding pattern (promoter %, FII, DII, mutual funds, public) from NSE quarterly filings
- Backtesting: simulate trading strategies on historical NSE price data (Yahoo Finance), with metrics including total return, CAGR, max drawdown, Sharpe ratio, and an equity curve

## Architecture

- **Go backend** (`main.go`, `internal/`) - fetches and scores market indicators, manages portfolio data, exposes a JSON API on `:8080`
  - `internal/api` - HTTP server (Gin), route registration, CORS middleware
  - `internal/indicators` - market indicator framework (NIFTY 50 PE)
  - `internal/mutualfund` - mutual fund search and holdings (mfapi.in + Yahoo Finance)
  - `internal/news` - RSS news aggregator (Economic Times, Moneycontrol, Business Standard) + in-memory stock news pipeline (`Store`)
  - `internal/nse` - NSE India HTTP client
  - `internal/stock` - stock price fetching via Yahoo Finance (used by the portfolio to auto-populate current value)
  - `internal/deepresearch` - per-stock deep research: annual reports via NSE (BSE fallback), financials fetched live from Yahoo Finance, supply chain extraction from PDFs, and shareholding pattern from NSE; all cached in Postgres via sqlc-generated queries
  - `internal/backtest` - backtesting engine: historical price fetching, strategy interface, trade simulation, and performance metrics. Strategies live alongside the engine (`buyhold.go`); add new ones by implementing `Strategy` and registering in `handler.go`
- **Python PDF parser** (`python/`) - long-running Flask HTTP service (`server.py`) on `:5001`; exposes `POST /parse` for supply chain (Related Party Transactions) extraction from annual report PDFs using `pdfplumber` with `pytesseract` OCR fallback for scanned PDFs
  - `internal/db` - PostgreSQL connection, startup migration (`schema.sql` embedded)
  - `internal/portfolio` - portfolio holdings CRUD
  - `internal/portfolio/db` - sqlc-generated type-safe query code (do not edit)
- **Next.js frontend** (`frontend/`) - indicators dashboard and portfolio management UI on `:3000`, with a live market news feed (tabbed: general market news and per-stock news) and individual stock view at `/stock/[symbol]` (current price, key metrics, quarterly/financial tables, documents, and RSS news feed)
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

Key environment variables (see `.env.example` for the full list):
- `PARSER_URL` (default `http://localhost:5001`) - location of the PDF parser service

## Screenshots

Capture screenshots of all frontend pages at desktop, tablet, and mobile viewports:

```bash
make screenshot   # uses the running docker-compose stack (auto-starts if needed)
```

Environment variables:
- `BASE_URL` — override server URL (default: `http://localhost:3001`)
- `START_SERVER=docker` — start the docker-compose stack first
- `START_SERVER=dev` — spawn the Next.js dev server instead
- `STOP_SERVER=1` — stop docker-compose when done

Screenshots are saved to `frontend/screenshots/`.

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

**Stock news pipeline:** A background ingester runs every 15 minutes, fetches the latest RSS articles, and matches each article to relevant stocks using keyword matching across ~65 major NSE-listed companies (Nifty 50 and others). Matched articles are pushed into the store automatically. The store deduplicates by article URL and normalises symbols to uppercase. Additional ingestion sources can also push news directly via `newsStore.Ingest(symbol, items)` or `newsStore.Replace(symbol, items)`.

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
| GET | `/api/portfolio/analyse` | Analyse mutual fund holdings: stock overlap, category concentration, and recommendations |

The analyse endpoint reads all `mutual_fund` holdings from the portfolio, fetches their stock-level holdings from Yahoo Finance via mfapi.in, then returns:
- Per-fund breakdown (fund house, category, top holdings)
- Stock overlap matrix (stocks held by more than one fund, with per-fund allocation %)
- Recommendations (consolidation suggestions, missing diversification)

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

### Stock Price

`GET /api/stock/:symbol/price` — fetch the current market price for an NSE stock symbol (e.g. `RELIANCE`). Appends `.NS` automatically for Yahoo Finance lookup.

```json
{
  "symbol": "RELIANCE.NS",
  "price": 1285.50,
  "currency": "INR",
  "short_name": "Reliance Industries Limited"
}
```

Used by the portfolio UI to auto-populate the current value field when adding or editing a stock holding.

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
  ],
  "shareholdingPattern": {
    "quarterEndDate": "31-Dec-2024",
    "category": {
      "promoterAndPromoterGroup": "50.33",
      "fii": "22.11",
      "dii": "12.44",
      "mutualFunds": "7.20",
      "publicAndOthers": "27.56"
    }
  }
}
```

The response also includes `financials` (P&L, Balance Sheet, Cash Flow, and per-share Highlights extracted from the PDF) and `_claudeFilled` (list of `"section.field"` keys that Claude populated when regex was insufficient). Supply chain results are cached in `supply_chain_store` and shareholding pattern in `shareholding_pattern_store` (30-day TTL), both via sqlc-generated queries keyed by symbol.

Uses `python/parse_pdf.py` (requires `pip install -r python/requirements.txt`; for scanned PDFs also needs `tesseract` and `poppler` system packages). Set `ANTHROPIC_API_KEY` to enable Claude gap-fill when fewer than 8 fields are extracted by regex.

### Backtesting

`POST /api/backtest` — simulate a trading strategy on historical daily price data for an NSE stock.

Request:

```json
{
  "symbol": "RELIANCE",
  "from": "2020-01-01",
  "to": "2025-01-01",
  "capital": 100000,
  "strategy": { "name": "buy_and_hold" }
}
```

Response:

```json
{
  "strategy": "buy_and_hold",
  "symbol": "RELIANCE.NS",
  "from": "2020-01-02",
  "to": "2024-12-31",
  "capital": 100000,
  "final_value": 142300,
  "trades": [
    { "entry_date": "2020-01-02", "entry_price": 1450.0, "exit_date": "2024-12-31", "exit_price": 2065.0, "shares": 68, "pnl": 41820.0, "return_pct": 42.4 }
  ],
  "equity_curve": [{ "date": "2020-01-02", "value": 100000 }, "..."],
  "metrics": {
    "total_return_pct": 42.4,
    "cagr_pct": 7.3,
    "max_drawdown_pct": 18.2,
    "sharpe_ratio": 1.1,
    "win_rate_pct": 100,
    "total_trades": 1
  }
}
```

Available strategies: `buy_and_hold`. New strategies implement the `Strategy` interface in `internal/backtest/` and are registered in `handler.go`.

Historical prices are fetched from Yahoo Finance (`.NS` suffix appended automatically for NSE symbols). No API key required.

## Indicators

| Indicator | Bullish | Neutral | Bearish |
|---|---|---|---|
| NIFTY 50 PE | < 20x | 20-25x | > 25x |
