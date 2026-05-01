# Yahoo Finance: probe findings

Findings from running `cmd/probes/yahoo` against `query1.finance.yahoo.com/v8/finance/chart`.
Run date: 2026-05-01. Range probed: 5 years.

> **Run it yourself:** `make probe-yahoo` (or `go run ./cmd/probes/yahoo`). Fixtures land in `testdata/yahoo/`.

---

## Summary

| Question | Answer |
|---|---|
| Do the major country indices return data via `v8/finance/chart`? | Yes, all 11 we tested. |
| Is a crumb required? | **No** for the chart endpoint. Probably yes for `quoteSummary` (TBD, separate probe). |
| Max history in a single call? | At least 5 years. Yahoo's `validRanges` advertises `1d, 5d, 1mo, 3mo, 6mo, 1y, 2y, 5y, 10y, ytd, max`. Use `period1` / `period2` Unix seconds rather than the canned `range` param for precision. |
| Are non-trading days included as nulls, or omitted? | **Omitted.** The `timestamp` array only contains trading days. A small number of nulls inside `close` still occur (1 to 5 per 5-year series), so the production decoder must treat `close` as `*float64`, not `float64`. |
| What does an unknown symbol return? | HTTP 404 with body `{"chart":{"result":null,"error":{"code":"Not Found","description":"No data found, symbol may be delisted"}}}`. The error path is structured and parseable. |
| User-Agent requirement? | A default Go UA gets rate-limited or blocked. We send a Chrome desktop UA and that works reliably. |
| Latency? | 30 to 100 ms per symbol on cold cache, single-threaded. |

---

## Symbol coverage

All probed indices returned 5 years of daily data on the first try, no crumb:

| Symbol  | Market | Currency | Timezone | Bars (5y) | Null closes |
|---|---|---|---|---|---|
| ^GSPC   | S&P 500            | USD | America/New_York   | 1255 | 0 |
| ^IXIC   | Nasdaq Composite   | USD | America/New_York   | 1255 | 0 |
| ^NSEI   | Nifty 50           | INR | Asia/Kolkata       | 1236 | 2 |
| ^BSESN  | BSE Sensex         | INR | Asia/Kolkata       | 1236 | 5 |
| ^N225   | Nikkei 225         | JPY | Asia/Tokyo         | 1222 | 1 |
| ^FTSE   | FTSE 100           | GBP | Europe/London      | 1261 | 1 |
| ^GDAXI  | DAX                | EUR | Europe/Berlin      | 1274 | 1 |
| ^HSI    | Hang Seng          | HKD | Asia/Hong_Kong     | 1227 | 1 |
| ^BVSP   | Bovespa            | BRL | America/Sao_Paulo  | 1247 | 0 |
| ^KS11   | KOSPI              | KRW | Asia/Seoul         | 1222 | 1 |
| ^AXJO   | ASX 200            | AUD | Australia/Sydney   | 1265 | 1 |

Bar count varies by ~50 across markets, consistent with each market's distinct holiday calendar.

---

## Response shape

The chart endpoint returns:

```jsonc
{
  "chart": {
    "result": [{
      "meta": {
        "currency": "USD",
        "symbol": "^GSPC",
        "exchangeName": "SNP",
        "fullExchangeName": "SNP",
        "instrumentType": "INDEX",
        "exchangeTimezoneName": "America/New_York",
        "timezone": "EDT",
        "gmtoffset": -14400,
        "firstTradeDate": -1325583000,
        "regularMarketPrice": 7209.01,
        "fiftyTwoWeekHigh": 7300.00,
        "fiftyTwoWeekLow": 5400.00,
        "validRanges": ["1d","5d","1mo","3mo","6mo","1y","2y","5y","10y","ytd","max"],
        // ... and more
      },
      "timestamp": [1620048600, 1620135000, ...],
      "indicators": {
        "quote": [{
          "open":   [4192.38, ...],
          "high":   [4218.91, ...],
          "low":    [4188.16, ...],
          "close":  [4192.66, ...],
          "volume": [3729360000, ...]
        }],
        "adjclose": [{ "adjclose": [4192.66, ...] }]
      }
    }],
    "error": null
  }
}
```

### Things to know about the shape

- **`timestamp` is at market open in UTC seconds**, not midnight. For NYSE, that is 13:30 UTC; for Tokyo, 00:00 UTC. Formatting via `time.Unix(ts, 0).UTC().Format("2006-01-02")` gives the correct trading date for every market we tested, because the open instant always falls within the same UTC date as the local trading date.
- **`close` and friends can contain `null`.** Decode as `*float64`, not `float64`. Treat null as "row missing" and skip, rather than zero.
- **`adjclose == close` for indices.** Indices are not adjusted for dividends because they are not investable; the `adjclose` array exists but matches `close`. We can ignore it for index data and use it for ETFs / stocks later.
- **`events` is absent for indices.** It is populated for ETFs and stocks when we ask for `events=div,split`. Not relevant for Phase 1.
- **`firstTradeDate` is a Unix timestamp** and can be negative for old indices (S&P 500 is `-1325583000`, December 1927). Useful as the lower bound when ingesting "all history".

---

## Hard requirements for the production client

1. **Set a browser-like `User-Agent`.** A default Go UA gets blocked.
2. **Decode prices as nullable.** `*float64` for OHLC, `*int64` for volume.
3. **Handle the 404-with-structured-error case** as a typed `ErrSymbolNotFound`, not a generic transport error.
4. **Use `period1` / `period2` Unix seconds**, not the `range=5y` shortcut. Backfill jobs need precise start dates.
5. **Use a `cookiejar`** even if no crumb is needed today: Yahoo has historically tightened auth without warning, and the cookie jar makes the eventual crumb path a one-line change.
6. **Rate-limit politely.** We saw no throttling at 11 sequential requests, but we did not stress-test. Start at 5 req/s and back off on 429.

---

## Open questions / follow-up probes

- **Does `quoteSummary` return `trailingPE` and `dividendYield` for index symbols, or only for tickers and ETFs?** Needs a separate probe (`cmd/probes/yahoo-quotesummary` or extend this one).
- **What is the practical rate limit on `chart`?** We did not push it. Phase 3 ingestion will need to know.
- **Is there a single bulk-quote endpoint for all current prices?** Would let the daily cron do one request instead of N. `query1.finance.yahoo.com/v7/finance/quote?symbols=...` is the candidate.

---

## Files

- Probe binary: `cmd/probes/yahoo/main.go`
- Fixtures: `testdata/yahoo/chart_<symbol>.json` (one per probed symbol, including the `^NOTREAL` error case)
