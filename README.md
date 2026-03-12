# marketview

A tool for Indian market investors to assess whether now is a good time to invest, based on live market indicators.

## Architecture

- **Go backend** (`main.go`, `internal/`) - fetches and scores market indicators, exposes a JSON API on `:8080`
  - `internal/api` - HTTP server, route registration, CORS middleware (Gin)
  - `internal/indicators` - market indicator framework (NIFTY 50 PE)
  - `internal/mutualfund` - mutual fund search and holdings (mfapi.in + Yahoo Finance)
  - `internal/news` - RSS news aggregator (Economic Times, Moneycontrol, Business Standard)
  - `internal/nse` - NSE India HTTP client
  - `internal/deepresearch` - per-stock deep research: annual reports via NSE (BSE fallback)
- **Next.js frontend** (`frontend/`) - dashboard at `/` showing all indicators color-coded by signal (bullish/neutral/bearish), auto-refreshes every 60 seconds, with a live market news feed and stock deep research at `/stock/[symbol]`

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

`GET /api/news` — returns up to 20 recent market news items aggregated from Economic Times, Moneycontrol, and Business Standard:

```json
[
  {
    "title": "Sensex rises 300 points...",
    "description": "...",
    "link": "https://...",
    "publishedAt": "2026-03-12T08:00:00Z",
    "source": "Economic Times"
  }
]
```

### Mutual Funds

`GET /api/mutual-fund/search?q={name}` — search for funds by name:

```json
[
  { "schemeCode": 119598, "schemeName": "Axis Bluechip Fund - Direct Growth" }
]
```

`GET /api/mutual-fund/{schemeCode}` — full fund details including stock holdings and allocation:

```json
{
  "schemeCode": 119598,
  "schemeName": "Axis Bluechip Fund - Direct Growth",
  "fundHouse": "Axis Mutual Fund",
  "schemeType": "Open Ended Schemes",
  "schemeCategory": "Equity Scheme - Large Cap Fund",
  "latestNAV": 56.78,
  "navDate": "11-03-2026",
  "navHistory": [{ "date": "11-03-2026", "nav": 56.78 }],
  "holdings": [
    { "name": "HDFC Bank Ltd", "symbol": "HDFCBANK.NS", "percentage": 9.52 }
  ],
  "stats": {
    "aum": 25000000000,
    "yield": 0.2,
    "ytdReturn": 3.5,
    "beta3Year": 0.95,
    "morningStarRating": 4,
    "equityPE": 25.3,
    "equityPB": 3.2,
    "stockAllocation": 95.0,
    "bondAllocation": 0.0,
    "cashAllocation": 5.0,
    "category": "India Fund Large-Cap"
  }
}
```

Holdings and stats are sourced from Yahoo Finance and may be absent for funds not listed there.

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
  "annualReportsSource": "NSE"
}
```

## Running

```bash
go run main.go        # starts backend on :8080
cd frontend && npm run dev  # starts Next.js on :3000
```

## Indicators

| Indicator | Bullish | Neutral | Bearish |
|---|---|---|---|
| NIFTY 50 PE | < 20x | 20-25x | > 25x |
