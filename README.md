# Marketview

A global investment indicator for retail investors.

Most retail dashboards show you a single market — yours. Marketview shows you the **whole world** in one place, so you can answer the question every long-term investor eventually asks:

> *"Where in the world should my next dollar go?"*

If India is overheated and the US is fairly priced, you should be able to see that at a glance. If Japanese small-caps just attracted $4B in ETF inflows, you should know. If three hedge funds rotated out of US tech into Brazilian energy last quarter, that's a signal you should not have to read a 13F filing to find.

---

## What it does

### 1. Global Market Dashboard

A single view of every major market in the world.

- **Country index heatmap** — S&P 500, Nasdaq, Nifty 50, Sensex, Nikkei 225, FTSE 100, DAX, CAC 40, Hang Seng, Shanghai Composite, KOSPI, Bovespa, ASX 200, and more. Returns over 1D / 1W / 1M / YTD / 1Y / 5Y.
- **Sector heatmap across regions** — compare US tech vs. India tech vs. China tech, or US energy vs. Brazilian energy, in a single grid.
- **Currency-adjusted returns** — a 20% Nifty rally means less to a USD investor if the rupee dropped 8%. Every return can be viewed in local currency or your home currency.
- **Valuation overlay** — P/E, Shiller CAPE, P/B, and dividend yield per country. Tells you whether a market is *cheap*, *fairly priced*, or *expensive*, not just whether it's up or down.
- **Macro context** — GDP growth, CPI, policy rate, 10Y yield, and currency trend overlaid on each market.

### 2. Capital Flows ("Where is the smart money going?")

A separate page tracking institutional money movement.

- **13F tracker** — quarterly holdings of large US hedge funds (Bridgewater, Berkshire, Tiger Global, Citadel, etc.) with quarter-over-quarter buys and sells.
- **ETF flows** — which country and sector ETFs are gaining or losing AUM, weekly.
- **FII / DII flows** — daily foreign vs. domestic institutional flows for the Indian market.
- **PE / VC deal tracker** — large private market deals by sector and region.
- **IPO pipeline** — upcoming and recent IPOs by country.
- **Insider transactions** — material insider buys and sells.

### 3. Compare & Decide

Tools to turn the data into a decision.

- **Side-by-side country compare** — pick two or three countries; see returns, valuation, flows, and macro head to head.
- **Attractiveness score** — a composite score per country, weighted across valuation, momentum, capital flows, and macro health. The "indicator" the product is named after.
- **Correlation matrix** — which markets actually diversify each other and which move together.
- **Backtest view** — *"$10k invested in this market 5 years ago, currency-adjusted, would now be worth…"*

### 4. Alerts & Signals *(later)*

- Notifications when a market enters extreme valuation territory.
- Notifications on flow reversals (e.g., FII flows turn net negative for the first time in N weeks).
- Earnings and macro event calendar by region.

---

## Tech stack

- **Backend** — Go (Gin), sqlc, PostgreSQL
- **Frontend** — Next.js (App Router), TypeScript, Tailwind
- **PDF parsing** — Python service (used for 13F and prospectus ingestion)
- **Infra** — Docker Compose for local dev

## Data sources (planned)

| Domain | Source |
|---|---|
| Country indices, equities | Yahoo Finance, Alpha Vantage |
| Hedge fund holdings | SEC EDGAR (13F filings) |
| Indian institutional flows | NSE / BSE FII-DII reports |
| Macro indicators | World Bank, IMF, FRED |
| ETF flows | ETF.com, issuer disclosures |
| FX rates | exchangerate.host / similar |
| IPOs | Exchange listings |

---

## Getting started

> Marketview is in active development. The setup below describes the intended local dev flow.

**Prerequisites**

- Docker and Docker Compose
- Go 1.22+
- Node.js 20+

**Setup**

```bash
git clone https://github.com/gsarmaonline/marketview.git
cd marketview
cp .env.example .env
docker compose up
```

Once running:

- Frontend: http://localhost:3001
- Backend API: http://localhost:8080
- PDF parser: http://localhost:5001

---

## Status

Early-stage. The MVP target covers the four sections above end-to-end. Contributions, ideas, and data-source suggestions are welcome.
