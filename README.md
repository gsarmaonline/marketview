# marketview

A tool for Indian market investors to assess whether now is a good time to invest, based on live market indicators.

## Architecture

- **Go backend** (`main.go`, `internal/`) - fetches and scores market indicators, exposes a JSON API on `:8080`
- **Next.js frontend** (`frontend/`) - consumes the API (pages not yet implemented)

## API

`GET /api/indicators` returns a JSON array of scored indicators:

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

## Running

```bash
go run main.go        # starts backend on :8080
cd frontend && npm run dev  # starts Next.js on :3000
```

## Indicators

| Indicator | Bullish | Neutral | Bearish |
|---|---|---|---|
| NIFTY 50 PE | < 20x | 20-25x | > 25x |
