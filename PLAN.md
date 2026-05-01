# Marketview Backend Implementation Plan

How we are going to build the backend. Companion to [ROADMAP.md](./ROADMAP.md) (product phases) and [README.md](./README.md) (vision).

**Scope of this document:** backend only. Frontend is deferred until the API is real.

---

## Stack

- **Language:** Go 1.22+
- **HTTP:** Gin
- **DB:** PostgreSQL, accessed via sqlc-generated queries (no ORM)
- **Migrations:** `golang-migrate`
- **Scheduling:** `robfig/cron` in-process for now; revisit if jobs grow
- **Testing:** stdlib `testing` + `testcontainers-go` for integration
- **Local infra:** `docker-compose.yml` (postgres only for now)

## Repo layout

```
cmd/
  server/          # marketview server (HTTP API)
  ingest/          # marketview ingest <job> (one-shot workers)
  probes/
    yahoo/         # live API probes (manual / weekly)
    edgar/
    fx/
internal/
  indices/         # vertical slice: store.go, service.go, handlers.go
  prices/
  fx/
  valuations/
  macro/
  flows/
  clock/           # injectable clock interface
  httpx/           # shared HTTP client helpers (UA, retry, rate limit)
db/
  migrations/      # golang-migrate files
  queries/         # sqlc input
  generated/       # sqlc output (committed)
docs/
  providers/       # one .md per third-party API with findings
testdata/
  yahoo/           # raw fixtures captured by the probes
  edgar/
  fx/
Makefile
sqlc.yaml
docker-compose.yml
```

## Core implementation rules

1. **One Go binary per concern.** `server` runs HTTP. `ingest <job>` runs a single ingestion to completion and exits. `probes/<x>` hits real APIs and saves fixtures.
2. **Vertical slices.** Each domain lives in its own `internal/<x>/` package with `store.go`, `service.go`, `handlers.go`. No shared "models" package.
3. **Each external API behind a small interface.** `PriceProvider`, `FilingProvider`, `FXProvider`. One method per use case. No premature generic ingestion framework.
4. **Idempotent ingestion, always.** Every writer uses `INSERT ... ON CONFLICT DO UPDATE`. Re-running yesterday's job must be a no-op.
5. **Precompute, do not compute per request.** Nightly job populates `index_returns(instrument_id, window, return_local, return_usd)`. The API just reads it.
6. **`clock.Clock` injected wherever time matters.** Returns, "today's close," "is the market open" — all of it must be testable.
7. **Provenance for every number.** Every row we write carries `source` and `source_fetched_at`. Trust is the moat.

---

## Phase A: Probes (do this first)

Before any production client code, verify the third-party APIs actually give us what we need. Each probe is a small Go binary. Each one saves real responses to `testdata/<provider>/` and writes findings to `docs/providers/<provider>.md`.

### A1. Yahoo Finance probe

**Build:** `cmd/probes/yahoo/main.go`

**Answer these questions:**

- [ ] Do `^GSPC, ^IXIC, ^NSEI, ^BSESN, ^N225, ^FTSE, ^GDAXI, ^HSI, ^BVSP, ^KS11, ^AXJO` all return data via `v8/finance/chart`?
- [ ] What is the max historical range we can request in one call?
- [ ] How long does crumb auth stay valid? Refresh strategy?
- [ ] Does `quoteSummary` return `trailingPE` / `dividendYield` for index symbols, or only for tickers?
- [ ] How are non-trading days represented in the timeseries?
- [ ] What does the response look like for a delisted or unknown symbol?

**Save fixtures to** `testdata/yahoo/chart_<symbol>.json`, `testdata/yahoo/quotesummary_<symbol>.json`.

### A2. EDGAR probe

**Build:** `cmd/probes/edgar/main.go`

**Answer these questions:**

- [ ] Fetch the latest 13F-HR for 3 funds with different styles (Berkshire CIK 0001067983, Bridgewater, Tiger Global). Schemas consistent?
- [ ] What is the practical rate limit with the required `User-Agent: Name email@x.com` header?
- [ ] What is the typical lag from quarter-end to filing availability?
- [ ] Can `data.sec.gov/submissions/CIK<n>.json` give us the latest 13F-HR URL without scraping index pages?
- [ ] Are share counts always shares, never dollar amounts? Any unit ambiguity?
- [ ] How do amendments (13F-HR/A) appear, and how should we handle supersession?

**Save fixtures to** `testdata/edgar/submissions_<cik>.json`, `testdata/edgar/13fhr_<cik>_<period>.xml`.

### A3. FX probe

**Build:** `cmd/probes/fx/main.go`

**Answer these questions:**

- [ ] Coverage of `INR, BRL, KRW, IDR, ZAR, MXN, TRY` against USD?
- [ ] Historical depth (need ≥5 years)?
- [ ] Daily granularity; how are weekends represented?
- [ ] Free-tier rate limits and TOS for our use case?
- [ ] Source of truth for the rate (mid? close?) and at what UTC time?

**Save fixtures to** `testdata/fx/historical_<base>_<quote>.json`.

**Probes are not thrown away.** They become `make test-live`, run manually or weekly, to detect upstream drift. CI uses the saved fixtures and never hits the real APIs.

---

## Phase B: Foundations

After probes confirm the APIs are usable.

- [ ] `go mod init github.com/gsarmaonline/marketview`
- [ ] Bring up `docker-compose.yml` with Postgres only
- [ ] Wire `golang-migrate` and `sqlc.yaml`; first migration creates an empty schema
- [ ] Makefile: `make up`, `make down`, `make migrate`, `make sqlc`, `make test`, `make test-live`, `make probe-<provider>`
- [ ] `cmd/server/main.go` with one `/healthz` route
- [ ] `internal/clock` with `Real` and `Fake` implementations
- [ ] `internal/httpx` with a shared client that sets UA, applies a configurable rate limit, retries on 429/5xx
- [ ] CI: GitHub Actions running `go vet`, `go test ./...`, and migration verification on every push

**Done when:** `make up && make test` is green from a fresh clone.

---

## Phase C: First vertical slice (one country index, end to end)

The smallest useful thing. Pick `^GSPC` and ride it end-to-end before generalising.

### Schema (first migration with real tables)

```sql
CREATE TABLE instruments (
  id            BIGSERIAL PRIMARY KEY,
  symbol        TEXT NOT NULL UNIQUE,
  name          TEXT NOT NULL,
  country       TEXT NOT NULL,
  currency      TEXT NOT NULL,
  kind          TEXT NOT NULL CHECK (kind IN ('country_index','sector_index')),
  source        TEXT NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE prices (
  instrument_id     BIGINT NOT NULL REFERENCES instruments(id),
  date              DATE NOT NULL,
  open              NUMERIC,
  high              NUMERIC,
  low               NUMERIC,
  close             NUMERIC NOT NULL,
  volume            BIGINT,
  source            TEXT NOT NULL,
  source_fetched_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (instrument_id, date)
);
```

### Implementation steps

- [ ] Migration for `instruments` and `prices`
- [ ] sqlc queries: `UpsertInstrument`, `UpsertPrice`, `ListInstruments`, `GetPricesByRange`
- [ ] `internal/prices/PriceProvider` interface
- [ ] `internal/prices/yahoo.Provider` implementing it (uses captured fixtures in tests, real API in `test-live`)
- [ ] `internal/prices/service.go` — `IngestDailyPrices(ctx, instrumentID, from, to)` with idempotent upsert
- [ ] `cmd/ingest/main.go` dispatch: `marketview ingest prices --symbol ^GSPC --from 2019-01-01`
- [ ] HTTP handlers in `internal/indices/handlers.go`:
  - `GET /api/indices`
  - `GET /api/indices/:id/prices?from=&to=`
- [ ] Seed migration inserts `^GSPC` as the first instrument
- [ ] Run the ingest command locally; verify 5+ years of S&P 500 daily prices in Postgres
- [ ] Hit the API endpoints and confirm the response shape

### Tests for this slice

- **Unit:** date-range math, response decoding from a saved fixture
- **Provider:** `yahoo.Provider` against a `httptest.Server` that serves `testdata/yahoo/chart_^GSPC.json`
- **Integration:** `prices.Service.IngestDailyPrices` against a real Postgres via testcontainers, asserting idempotency by running it twice and checking row counts

**Done when:** one country index has 5+ years of daily prices in Postgres, the API returns them, and re-running ingestion does not duplicate rows.

---

## Phase D: Generalise the slice

Repeat the Phase C shape for the rest of Phase 1 of the roadmap.

- [ ] Seed remaining country indices via migration (~20 total)
- [ ] One ingest job iterates all instruments
- [ ] `index_returns` table + nightly precompute job for `1D / 1W / 1M / YTD / 1Y / 5Y` returns
- [ ] `GET /api/indices/returns?range=YTD` reads from the precomputed table
- [ ] Add scheduler entry that runs the prices ingest daily after US close

**Done when:** every seeded index has up-to-date prices and a precomputed returns row for every standard window.

---

## Testing strategy (summary)

| Layer | What it covers | Speed | Hits network? | Hits DB? |
|---|---|---|---|---|
| Unit | Pure functions: returns, FX conversion, scoring | <1s | No | No |
| Provider | API clients via `httptest.Server` + saved fixtures | <2s | No (fixture replay) | No |
| Integration | Stores + services against real Postgres (testcontainers) | ~10s | No | Yes |
| Live (probes) | Real third-party APIs, drift detection | varies | Yes | No |

**Hard rules:**

- No DB mocking. We use Postgres in tests because we use Postgres in prod.
- CI never hits real third-party APIs. Only `make test-live` does.
- Every ingestion test must include an idempotency assertion (run twice, expect identical state).
- Every provider client must have a fixture-driven test for at least one happy path and one error path (404, malformed body).

---

## What we are explicitly not doing yet

- Frontend. Deferred until the API is real.
- Authentication. The API is open inside our network until Phase 6 of the roadmap.
- Caching layer (Redis). The precomputed tables are the cache.
- Background queues (Temporal, Asynq). One-shot ingest binaries are sufficient.
- Multi-region deployment. Single Postgres, single server until usage justifies more.
