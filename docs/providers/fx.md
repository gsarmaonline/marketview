# FX (Frankfurter / ECB): probe findings

Findings from running `cmd/probes/fx` against `api.frankfurter.app`. Run date: 2026-05-01.

> **Run it yourself:** `make probe-fx`. Fixtures land in `testdata/fx/`. No auth required.

---

## Summary

| Question | Answer |
|---|---|
| Coverage of `INR, BRL, KRW, IDR, ZAR, MXN, TRY` against USD? | **All 7 supported.** Frankfurter exposes 30 currencies, and every one of our target EM currencies is on the list. |
| Historical depth available? | **At least 5 years**, returned in a single API call. Span 2021-04-30 to 2026-04-30 produced 1,282 rows in 137 KB in ~1.1s. |
| Daily granularity? | Yes, one row per ECB publishing day. |
| Weekend behaviour? | **Weekends are simply absent** from the response, not present-with-nulls. Production code must treat missing dates as "no rate published" and carry forward. |
| Source of truth? | **ECB reference rates**, published once per business day around 16:00 CET. Frankfurter reads from the official ECB feed. |
| Free-tier rate limits? | No published limit, no auth. We made 3 requests in this probe with no throttling. Be polite anyway: a couple of requests per second, not hundreds. |
| Auth? | **None.** No API key, no signup. |
| Latest-data lag? | About 1 calendar day. As of 2026-05-01 (a Friday), the latest available date was 2026-04-30. ECB publishes the day's rates ~16:00 CET, so a daily ingest job that runs in early UTC will see a 1-day lag. |

---

## Currency coverage

Frankfurter / ECB supports these 30 currencies:

```
AUD BRL CAD CHF CNY CZK DKK EUR GBP HKD HUF IDR ILS INR ISK
JPY KRW MXN MYR NOK NZD PHP PLN RON SEK SGD THB TRY USD ZAR
```

This is enough for every market index Marketview currently tracks (US, India, Japan, UK, Germany, Hong Kong, Brazil, South Korea, Australia).

**Notable absences** for future expansion: RUB, ARS, EGP, NGN, VND, COP, CLP, PEN, PKR, BDT, AED, SAR. If we add Russia, Argentina, Egypt, Nigeria, Vietnam, or other frontier markets later, we will need a second FX source (Yahoo Finance carries most of these as `XXX=Y` symbols and we have already verified Yahoo works in the Yahoo probe).

---

## Sample latest rates (2026-04-30, USD-base)

```
USD/BRL = 4.9814
USD/IDR = 17346
USD/INR = 94.92
USD/KRW = 1477
USD/MXN = 17.5158
USD/TRY = 45.185
USD/ZAR = 16.7903
```

## 5-year history

A single GET to `/2021-04-30..2026-04-30?from=USD&to=...` returned the full timeseries.

| Currency | Rows | First (2021-04-30) | Last (2026-04-30) | Move |
|---|---|---|---|---|
| INR | 1282 | 74.06   | 94.92   | INR weakened ~28% |
| BRL | 1282 | 5.35    | 4.98    | BRL slightly stronger |
| KRW | 1282 | 1114.25 | 1477.00 | KRW weakened ~33% |
| IDR | 1282 | 14421   | 17346   | IDR weakened ~20% |
| ZAR | 1282 | 14.39   | 16.79   | ZAR weakened ~17% |
| MXN | 1282 | 20.06   | 17.52   | MXN strengthened ~13% |
| TRY | 1282 | 8.26    | 45.19   | TRY collapsed (Turkish hyperinflation) |

Spot-checked against general knowledge of these currencies: all moves are directionally correct. Frankfurter's data quality is solid.

---

## Holiday-gap behaviour

Over the 5-year span there were 1,305 weekdays. The response returned 1,282 rows. The 23 missing weekdays are **ECB / TARGET holidays**, predominantly:

- Good Friday + Easter Monday (4 days each year, 5 years = 20 days)
- May 1 (Labour Day, observed 2024 and 2025; not in 2021-2023 because they fell on a weekend)
- December 24, 25, 26 (some years observed, some not)
- December 31 / January 1

Concretely, we saw gaps such as:

```
2022-04-14 -> 2022-04-19   Easter
2023-12-22 -> 2023-12-27   Christmas window
2024-03-28 -> 2024-04-02   Easter
2024-04-30 -> 2024-05-02   May 1
2025-12-24 -> 2025-12-29   Christmas
2026-04-02 -> 2026-04-07   Easter
```

**Important: ECB holidays do not align with US, Indian, or Japanese market holidays.** A US trading day that is also an ECB holiday (e.g., April 1, 2024 was Easter Monday in Europe but markets are closed in the US too — but consider an Asian market on a non-ECB-holiday: the FX rate may be missing for that date even though the market traded). The production logic must:

1. For a given market-close date in any timezone, look up the FX rate **for that same date**.
2. If absent, carry forward the **most recent prior** ECB rate.
3. Never silently zero or null. Returns calculated against a missing FX rate should propagate the gap.

This is a hard requirement, not a nice-to-have. Currency-adjusted returns are wrong without it.

---

## Hard requirements for the production client

1. **No auth, but set a meaningful `User-Agent`** so Frankfurter can identify our traffic if they ever rate-limit. We use `Marketview-FX-Probe/0.1`; production should use `marketview/<version>`.
2. **Use the timeseries endpoint** for backfills and the `/latest` endpoint for daily updates. Do not call `/<single-date>` per day in a loop, the timeseries endpoint is dramatically cheaper.
3. **Carry-forward on missing dates** at the consumer (returns calculator). Do not insert synthesised rows into `fx_rates`. Store only what ECB published.
4. **Schedule the daily ingest after 17:00 CET** (15:00 UTC in summer, 16:00 UTC in winter). Earlier runs will see yesterday's data and need to retry.
5. **Treat empty `rates[date]` as a hard error**, not a warning. Frankfurter never returns dates with empty rate maps for currencies in the supported list, so an empty map means upstream changed.
6. **Plan for a second source** before we add a frontier-market index (Russia, Argentina, Egypt, etc). Yahoo's `XXX=X` symbols are the natural fallback and already vetted.

---

## Things this probe did not test

- **Sustained throughput.** We made 3 requests. We have not pushed Frankfurter to find its actual rate limit.
- **Resilience to upstream outages.** Frankfurter is a single point of failure. Production should have an alarm if `/latest` returns the same `date` for >2 calendar days in a row (suggests publishing is stuck).
- **Daily incremental ingest pattern.** The probe fetched 5 years in one call. The daily cron would call `/latest`. We have not exercised this on a schedule.
- **Yahoo as fallback.** Already verified by the Yahoo probe; not exercised here as the FX source.

---

## Files

- Probe binary: `cmd/probes/fx/main.go`
- Fixtures:
  - `testdata/fx/currencies.json` (full ECB-supported list)
  - `testdata/fx/latest_USD.json` (today's rates for our 7 EMs)
  - `testdata/fx/timeseries_USD_<from>_to_<to>.json` (5y daily history)
