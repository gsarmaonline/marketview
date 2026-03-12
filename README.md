# marketview

A tool for Indian market investors to assess whether now is a good time to invest, based on live market indicators.

## Architecture

- **Go backend** (`main.go`, `internal/`) - fetches and scores market indicators, exposes a JSON API on `:8080`
  - `internal/api` - HTTP server, route registration, CORS middleware
  - `internal/indicators` - market indicator framework (NIFTY 50 PE)
  - `internal/mutualfund` - mutual fund search and holdings (mfapi.in + Yahoo Finance)
  - `internal/nse` - NSE India HTTP client
- **Next.js frontend** (`frontend/`) - consumes the API (pages not yet implemented)

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

## Running

```bash
go run main.go        # starts backend on :8080
cd frontend && npm run dev  # starts Next.js on :3000
```

## Indicators

| Indicator | Bullish | Neutral | Bearish |
|---|---|---|---|
| NIFTY 50 PE | < 20x | 20-25x | > 25x |
