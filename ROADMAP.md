# Marketview Roadmap

A phased plan to build Marketview from an empty repo to the full MVP described in [README.md](./README.md).

Each phase delivers something useful on its own: the product is shippable at the end of every phase, not just at the end. Effort markers: **S** (small, ~days), **M** (medium, ~weeks), **L** (large, multi-week).

---

## Phase 0: Foundations  *(S)*

Get the skeleton running before any market logic is written.

- [ ] Scaffold Go backend (Gin, sqlc, Postgres) with a single `/healthz` endpoint
- [ ] Scaffold Next.js frontend (App Router, TypeScript, Tailwind) with one placeholder page
- [ ] Wire `docker-compose.yml`: backend, frontend, postgres, pdf-parser
- [ ] Makefile targets: `make up`, `make down`, `make migrate`, `make sqlc`, `make test`
- [ ] CI: lint + test on every push (GitHub Actions)
- [ ] Migration tooling (golang-migrate or goose) wired into the Makefile

**Done when:** `make up` brings the whole stack up, frontend can call backend, CI is green.

---

## Phase 1: Global Market Dashboard, v1  *(M)*

The smallest version of the core "where to invest" view. Local-currency only, no valuation, no macro yet.

### Backend
- [ ] `instruments` table: country indices (Nifty 50, S&P 500, Nikkei, etc.). Seed ~20 major indices.
- [ ] `prices` table: daily OHLC per instrument
- [ ] Ingestion worker: pull daily prices from Yahoo Finance for all seeded indices
- [ ] Scheduler (cron in-process or `robfig/cron`) to run ingestion daily after market close
- [ ] API: `GET /api/indices` (list), `GET /api/indices/:id/prices?range=1Y` (timeseries)
- [ ] API: `GET /api/indices/returns?range=YTD` (precomputed returns over standard windows)

### Frontend
- [ ] World heatmap page: grid of country indices with color-coded YTD return
- [ ] Range toggle: 1D / 1W / 1M / YTD / 1Y / 5Y
- [ ] Index detail page with a price chart (Recharts or lightweight-charts)

**Done when:** a retail user can land on the homepage and see at a glance which markets are up or down over a chosen window.

---

## Phase 2: Currency Adjustment + Sector Heatmap  *(M)*

Make the dashboard honest for cross-border investors and add the second cross-cut.

### Currency adjustment
- [ ] `fx_rates` table: daily USD/local rates for every market we cover
- [ ] Ingestion: exchangerate.host (or similar)
- [ ] Returns API gains a `currency=USD|local|home` parameter
- [ ] Frontend toggle: "Local currency" vs. "USD"

### Sector heatmap
- [ ] Extend `instruments` to cover sector indices (US tech, India tech, US energy, etc.) using sector ETFs as proxies where direct indices don't exist
- [ ] New page: sector × region grid

**Done when:** a USD-based retail investor sees true returns, and can compare US tech vs. India tech vs. China tech in one view.

---

## Phase 3: Valuation + Macro Overlays  *(M)*

Now the dashboard answers "is this market expensive?" not just "is it up?"

### Valuation
- [ ] `valuations` table per index: P/E, P/B, dividend yield, CAPE
- [ ] Ingestion: a mix of Yahoo Finance, issuer disclosures, and (for CAPE) Robert Shiller's published data + per-country approximations
- [ ] Heatmap gains a "valuation mode" toggle: returns vs. P/E vs. CAPE

### Macro
- [ ] `macro_indicators` table: GDP growth, CPI, policy rate, 10Y yield per country
- [ ] Ingestion: World Bank + IMF + FRED APIs (free)
- [ ] Index detail page shows a macro panel

**Done when:** the user can see, side by side, "India is up 18% YTD but trades at a 24x P/E with 5.2% inflation" without leaving the page.

---

## Phase 4: Capital Flows Page  *(L)*

The "smart money" view. Largest data-engineering phase.

### 13F tracker
- [ ] `funds`, `fund_holdings`, `fund_holdings_changes` tables
- [ ] EDGAR ingestion worker: fetch 13F-HR filings quarterly for a curated list (~30 funds)
- [ ] Parse 13F XML into structured holdings (no PDF parsing needed for 13Fs, they are XML)
- [ ] Compute QoQ buys/sells/new positions/exits
- [ ] Page: top funds, top buys, top sells, top concentrated positions

### ETF flows
- [ ] `etfs`, `etf_aum_history` tables
- [ ] Weekly ingestion of AUM by country/sector ETF
- [ ] Page: weekly flow leaderboard by region and sector

### India FII/DII
- [ ] Daily scrape of NSE/BSE FII-DII reports
- [ ] Chart: rolling FII vs. DII flows
- [ ] Visible on the India country detail page

### IPO pipeline + insider transactions
- [ ] IPO calendar by country (exchange listings + EDGAR S-1s for the US)
- [ ] Insider transactions for US (Form 4 from EDGAR)

### PE/VC deal tracker (last in this phase)
- [ ] Likely starts as a manual curation table; full automation deferred unless a free source emerges

**Done when:** there is a /flows page that shows, at minimum, 13F changes + ETF flows + India FII/DII for the most recent reporting period.

---

## Phase 5: Compare & Decide  *(M)*

Turn data into decisions.

- [ ] **Side-by-side compare** page: pick 2–3 countries, render returns / valuation / flows / macro head to head
- [ ] **Attractiveness score**: composite per country, weighted blend of
  - valuation z-score (low CAPE relative to its history is good)
  - momentum (trailing 6M return)
  - flow signal (net institutional flow direction)
  - macro health (real growth minus inflation, currency stability)
  - Document the formula transparently on the page (retail trust)
- [ ] **Correlation matrix** of country index returns over a configurable window
- [ ] **Backtest widget**: "$X invested N years ago in this market, currency-adjusted, would now be worth $Y"

**Done when:** the homepage shows attractiveness scores, and there is a /compare page.

---

## Phase 6: Accounts, Watchlists, Alerts  *(M)*

Make it sticky for retail.

- [ ] User auth (email magic link or OAuth)
- [ ] Watchlists: track specific countries, sectors, or funds
- [ ] Email/push alerts:
  - Country crosses an attractiveness threshold
  - Flow direction reverses (e.g., FII flips negative for the first time in N weeks)
  - Valuation hits 90th-percentile of its 10Y history
- [ ] Earnings + macro event calendar by region

**Done when:** users can sign up, save watchlists, and receive at least one type of alert.

---

## Phase 7: Polish + Launch  *(M)*

- [ ] Mobile-responsive pass on every page
- [ ] Loading and empty states
- [ ] Onboarding tour: "what is Marketview, in 30 seconds"
- [ ] SEO (server-rendered country pages with valuation snapshots)
- [ ] Public landing page
- [ ] Analytics (PostHog or Plausible)
- [ ] Production deploy (Fly.io, Render, or similar) with managed Postgres

---

## Cross-cutting concerns (apply throughout)

- **Data quality**: every ingestion job logs row counts and validates against last-good values; alert on >5σ deviations
- **Caching**: heavy aggregations (returns, attractiveness scores) precomputed nightly into materialized tables, not computed per request
- **Rate limits**: respect Yahoo, Alpha Vantage, and EDGAR rate limits. Use polite User-Agents on EDGAR (it is a hard requirement)
- **Provenance**: every number on every page links to a "source" tooltip. Trust is the moat for a retail product
- **Disclaimer**: not investment advice. Visible in the footer from day one

---

## Explicit non-goals (for now)

- Real-time / intraday data. End-of-day is fine for MVP and dramatically simpler.
- Order placement / brokerage integration.
- Crypto.
- Individual-stock screening (we are about *markets*, not stock picking).
- Custom portfolio analytics. The repo had this previously and was deliberately reset; do not re-introduce it inside Marketview.
