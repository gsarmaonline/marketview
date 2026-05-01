# SEC EDGAR: probe findings

Findings from running `cmd/probes/edgar` against `data.sec.gov` (submissions JSON) and `www.sec.gov/Archives` (filing artefacts). Run date: 2026-05-01.

> **Run it yourself:** `EDGAR_CONTACT_EMAIL=you@example.com make probe-edgar`. Fixtures land in `testdata/edgar/`.

---

## Summary

| Question | Answer |
|---|---|
| Schema consistent across funds with different styles? | **Yes**, with namespace caveats (see below). All four probed funds parsed cleanly with the same struct. |
| Practical rate limit with mandatory `User-Agent`? | We ran at 5 req/s with no throttling, no 429s. SEC's documented ceiling is 10 req/s. |
| Lag from quarter-end to 13F filing? | **43 to 48 days** across the four funds for Q4 2025. The 45-day deadline is real and most funds file close to it. |
| Can submissions JSON give us the latest 13F-HR without scraping HTML? | **Yes for the metadata** (form, date, accession). We still need a second call to `Archives/.../index.json` to find the info table filename, because the info table file is not named consistently. |
| Are share counts always shares? | All probed filings show `sshPrnamtType=SH` (shares) for 100% of holdings. `PRN` (principal amount) does exist in the schema for bonds but is rarely used in 13Fs (which are equity-focused). |
| How do amendments appear? | As `13F-HR/A` in `recent.form`. Bridgewater had 15 amendments in their last 100 filings, Berkshire had 5. The original 13F-HR stays in the record; production code needs an explicit supersession rule (latest filing for a given `reportDate` wins). |

---

## Probed funds

| CIK         | Fund                          | Recent filings | 13F-HR | 13F-HR/A | Latest report | Filed      | Lag | Holdings | Total value (raw, $1s) |
|---|---|---|---|---|---|---|---|---|---|
| 0001067983  | Berkshire Hathaway            | 1000           | 39     | 5        | 2025-12-31    | 2026-02-17 | 48d | 110      | $274.16B |
| 0001350694  | Bridgewater Associates        | 100            | 81     | 15       | 2025-12-31    | 2026-02-13 | 44d | 1,040    | $27.42B  |
| 0001167483  | Tiger Global Management       | 525            | 98     | 2        | 2025-12-31    | 2026-02-17 | 48d | 54       | $29.71B  |
| 0001037389  | Renaissance Technologies      | 1071           | 25     | 0        | 2025-12-31    | 2026-02-12 | 43d | 3,185    | $64.46B  |

Holdings counts span ~50 to ~3,200, so the production parser needs to be efficient at the high end and not assume a small portfolio.

---

## Three-step fetch pattern

For each fund, the production client must do:

```
1. GET https://data.sec.gov/submissions/CIK<10-digit-padded>.json
   -> filter recent.form == "13F-HR" / "13F-HR/A", pick latest by filingDate
2. GET https://www.sec.gov/Archives/edgar/data/<unpadded-cik>/<accession-no-dashes>/index.json
   -> find the info-table XML filename
3. GET https://www.sec.gov/Archives/edgar/data/<unpadded-cik>/<accession-no-dashes>/<filename>
   -> parse the holdings information table
```

Path quirks:

- **CIK formatting differs by host.** `data.sec.gov` wants `CIK0001067983` (10-digit zero-padded). `www.sec.gov/Archives/edgar/data/` wants `1067983` (unpadded).
- **Accession number formatting differs by use site.** Submissions JSON gives `0001193125-26-054580` with dashes. The Archives directory name is `000119312526054580` (no dashes). Strip dashes when building the path.
- **Accession-number prefix is the filing agent's CIK, not the subject's.** Berkshire's filing was made by Donnelley Financial Solutions (`0001193125`), so the accession starts with that CIK rather than Berkshire's. Do not infer the subject CIK from the accession.

---

## Info table XML quirks

All four fixtures live at `testdata/edgar/infotable_<cik>_<accession>.xml`. They share the same `informationTable / infoTable` schema but differ in surface details:

| Fund | Namespace prefix | Encoding declaration | Extra fields | Filename |
|---|---|---|---|---|
| Berkshire     | default `xmlns="...thirteenf/informationtable"` | none           | none           | `50240.xml` (no convention) |
| Bridgewater   | `ns1:` prefix on every element                  | `utf-8` decl   | adds `figi`    | accession-based |
| Tiger Global  | default + `xsi:schemaLocation`                  | `<?xml version="1.0" ?>` | none | accession-based |
| Renaissance   | default                                         | `UTF-8` decl   | none           | accession-based |

**Hard requirement:** the production parser must ignore namespaces entirely. The probe sets `xml.Decoder.Strict = false` and decodes with a struct whose tags name the local elements (`xml:"infoTable"`, `xml:"nameOfIssuer"`, etc.), without binding to any namespace URI. This handled all four variants without changes.

**Filename heuristic:** the info table is sometimes named `<something>infotable.xml` or `<accession>-information_table.xml`, but Berkshire's was just `50240.xml`. Heuristic that worked: prefer files whose name contains `infotable` / `informationtable`, otherwise pick the largest `.xml` that is not the `primaryDocument` referenced in the submissions response.

---

## The value-units gotcha

13F `<value>` units changed:

- **Pre-2023-01-03 filings:** values are in **$1000s** (thousands of dollars).
- **Post-2023-01-03 filings:** values are in **whole dollars** ($1s).

This is invisible to the parser, the magnitude alone is not a reliable discriminator (a $30B post-2023 portfolio in $1s and a $30M pre-2023 portfolio in $1000s look identical at the top level). **Use the filing date as the discriminator.**

The probe initially used a magnitude-based heuristic and got it wrong for three of the four funds. It now uses filing date and labels output `(post-2023 schema)` vs `(pre-2023 schema)`. Production code must do the same.

For historical backfills crossing the 2023 cutoff, normalise everything to whole dollars at ingest time and store a single `value_usd` column. Do not push the unit ambiguity into downstream code.

---

## Things this probe did not test

- **Pagination of older filings.** Berkshire and Renaissance both have additional filings in `filings.files[]` beyond the `recent` block. The probe only reads `recent`. For Phase 4 backfill of 5+ years of 13Fs we will need to follow the additional file URLs.
- **Sustained rate-limit behaviour.** We made a few dozen requests at 5 req/s with no issues. We have not pushed the 10 req/s ceiling or run for an hour straight. Production ingestion should retain the 200ms throttle and add 429-aware exponential backoff.
- **Older schema versions** (pre-2013 filings used a different XSD). Phase 4 of the roadmap may decide to ingest only post-2013 to dodge this.
- **Options positions.** Zero of the four probed filings had `putCall` populated. Some funds (e.g. quant shops, options arbs) will have non-empty `putCall`; the production schema must allow it.
- **Bond positions** (`sshPrnamtType=PRN`). Zero in our probe; the schema allows it; we may simply skip non-`SH` rows in Phase 4.

---

## Hard requirements for the production client

1. **Mandatory `User-Agent: <Name> <email>`**. Read `EDGAR_CONTACT_EMAIL` from env or config. Failing fast on missing email is correct; SEC will block silently otherwise.
2. **Rate limit at 5 req/s** by default with 429-aware backoff. SEC's published ceiling is 10/s but politeness costs us nothing.
3. **Three-step fetch:** submissions -> filing index -> info table. Cache submissions JSON aggressively (it changes only when the fund files something new).
4. **XML decoder must be namespace-blind.** `xml.Decoder` with `Strict = false` and struct tags using local names handles all four observed variants.
5. **Use filing date to determine value units.** Pre-2023-01-03 = $1000s, on/after = $1s. Normalise to $1s at write time.
6. **Amendment supersession:** for a given `(cik, reportDate)`, the latest `filingDate` wins, regardless of whether it is `13F-HR` or `13F-HR/A`.
7. **Do not infer subject CIK from accession number prefix.** The prefix is the filing agent.

---

## Files

- Probe binary: `cmd/probes/edgar/main.go`
- Fixtures (per fund):
  - `testdata/edgar/submissions_<cik>.json`
  - `testdata/edgar/index_<cik>_<accession>.json`
  - `testdata/edgar/infotable_<cik>_<accession>.xml`
