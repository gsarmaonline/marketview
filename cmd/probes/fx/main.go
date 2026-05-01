// FX probe.
//
// Hits api.frankfurter.app (free, no auth, ECB reference rates) to verify
// coverage and historical depth for the EM currencies Marketview needs.
// Saves the supported-currency list, latest rates, and a 5-year USD-base
// timeseries to testdata/fx, and prints findings that answer the FX
// questions in PLAN.md.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	frankfurterHost = "https://api.frankfurter.app"
	userAgent       = "Marketview-FX-Probe/0.1 (+https://github.com/gsarmaonline/marketview)"
)

// Default EM-heavy currency list lifted from PLAN.md.
var defaultCurrencies = []string{"INR", "BRL", "KRW", "IDR", "ZAR", "MXN", "TRY"}

type currenciesResponse map[string]string

type latestResponse struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Date   string             `json:"date"`
	Rates  map[string]float64 `json:"rates"`
}

type timeseriesResponse struct {
	Amount    float64                       `json:"amount"`
	Base      string                        `json:"base"`
	StartDate string                        `json:"start_date"`
	EndDate   string                        `json:"end_date"`
	Rates     map[string]map[string]float64 `json:"rates"`
}

type probeClient struct {
	http *http.Client
}

func newClient() *probeClient {
	return &probeClient{http: &http.Client{Timeout: 30 * time.Second}}
}

func (c *probeClient) get(ctx context.Context, rawURL string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return body, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return body, resp.StatusCode, fmt.Errorf("status %d for %s: %s", resp.StatusCode, rawURL, truncate(string(body), 200))
	}
	return body, resp.StatusCode, nil
}

func writeFixture(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func main() {
	base := flag.String("base", "USD", "base currency")
	currenciesFlag := flag.String("currencies", strings.Join(defaultCurrencies, ","), "comma-separated quote currencies to probe")
	years := flag.Int("years", 5, "history depth in years")
	outDir := flag.String("out", "testdata/fx", "fixture output directory")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		die(err)
	}

	c := newClient()
	ctx := context.Background()
	wanted := strings.Split(*currenciesFlag, ",")
	for i := range wanted {
		wanted[i] = strings.ToUpper(strings.TrimSpace(wanted[i]))
	}

	// 1. /currencies: full supported list.
	currURL := frankfurterHost + "/currencies"
	currBody, _, err := c.get(ctx, currURL)
	if err != nil {
		die(err)
	}
	currPath := filepath.Join(*outDir, "currencies.json")
	_ = writeFixture(currPath, currBody)

	var supported currenciesResponse
	if err := json.Unmarshal(currBody, &supported); err != nil {
		die(fmt.Errorf("parse /currencies: %w", err))
	}
	missing := []string{}
	for _, w := range wanted {
		if _, ok := supported[w]; !ok {
			missing = append(missing, w)
		}
	}
	fmt.Printf("== /currencies ==\n")
	fmt.Printf("  supported total : %d\n", len(supported))
	fmt.Printf("  requested wanted: %v\n", wanted)
	if len(missing) > 0 {
		fmt.Printf("  MISSING         : %v (consider Yahoo fallback for these)\n", missing)
	} else {
		fmt.Printf("  coverage        : all wanted currencies are supported\n")
	}
	fmt.Printf("  fixture         : %s\n\n", currPath)

	// 2. /latest: today's rates for the wanted set.
	latestURL := fmt.Sprintf("%s/latest?from=%s&to=%s",
		frankfurterHost, url.QueryEscape(*base), url.QueryEscape(strings.Join(wanted, ",")))
	latestBody, _, err := c.get(ctx, latestURL)
	if err != nil {
		fmt.Println("WARN: /latest failed:", err)
	}
	latestPath := filepath.Join(*outDir, fmt.Sprintf("latest_%s.json", *base))
	_ = writeFixture(latestPath, latestBody)

	var latest latestResponse
	if err := json.Unmarshal(latestBody, &latest); err == nil {
		fmt.Printf("== /latest ==\n")
		fmt.Printf("  base=%s  date=%s\n", latest.Base, latest.Date)
		// Determine staleness.
		if d, err := time.Parse("2006-01-02", latest.Date); err == nil {
			age := int(time.Since(d) / (24 * time.Hour))
			fmt.Printf("  age vs today    : %d day(s)  (ECB does not publish weekends/holidays)\n", age)
		}
		// Print rates in stable order.
		ks := make([]string, 0, len(latest.Rates))
		for k := range latest.Rates {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		for _, k := range ks {
			fmt.Printf("  %s/%s = %g\n", *base, k, latest.Rates[k])
		}
		fmt.Printf("  fixture         : %s\n\n", latestPath)
	}

	// 3. /<from>..<to> timeseries for 5y depth.
	to := time.Now().UTC()
	from := to.AddDate(-*years, 0, 0)
	tsURL := fmt.Sprintf("%s/%s..%s?from=%s&to=%s",
		frankfurterHost,
		from.Format("2006-01-02"), to.Format("2006-01-02"),
		url.QueryEscape(*base), url.QueryEscape(strings.Join(wanted, ",")))
	tsStart := time.Now()
	tsBody, _, err := c.get(ctx, tsURL)
	tsDur := time.Since(tsStart)
	if err != nil {
		die(err)
	}
	tsPath := filepath.Join(*outDir, fmt.Sprintf("timeseries_%s_%s_to_%s.json",
		*base, from.Format("2006-01-02"), to.Format("2006-01-02")))
	_ = writeFixture(tsPath, tsBody)

	var ts timeseriesResponse
	if err := json.Unmarshal(tsBody, &ts); err != nil {
		die(fmt.Errorf("parse timeseries: %w", err))
	}

	fmt.Printf("== /timeseries (%s..%s) ==\n", ts.StartDate, ts.EndDate)
	fmt.Printf("  base=%s  rows=%d  bytes=%d  fetch=%dms\n",
		ts.Base, len(ts.Rates), len(tsBody), tsDur.Milliseconds())

	if len(ts.Rates) > 0 {
		// Date span and gap analysis.
		dates := make([]string, 0, len(ts.Rates))
		for d := range ts.Rates {
			dates = append(dates, d)
		}
		sort.Strings(dates)
		first, _ := time.Parse("2006-01-02", dates[0])
		last, _ := time.Parse("2006-01-02", dates[len(dates)-1])
		span := int(last.Sub(first)/(24*time.Hour)) + 1
		businessDays := countBusinessDays(first, last)
		gaps := businessDays - len(dates)
		fmt.Printf("  span days       : %d  (first=%s, last=%s)\n", span, dates[0], dates[len(dates)-1])
		fmt.Printf("  business days   : %d  (Mon-Fri in span)\n", businessDays)
		fmt.Printf("  rows returned   : %d\n", len(dates))
		fmt.Printf("  weekday gaps    : %d  (likely ECB / TARGET holidays)\n", gaps)
	}

	// Per-currency null / coverage check.
	type ccyStats struct {
		Currency  string
		Present   int
		Missing   int
		FirstSeen string
		LastSeen  string
		FirstRate float64
		LastRate  float64
	}
	stats := make(map[string]*ccyStats)
	for _, w := range wanted {
		stats[w] = &ccyStats{Currency: w}
	}
	dateKeys := make([]string, 0, len(ts.Rates))
	for d := range ts.Rates {
		dateKeys = append(dateKeys, d)
	}
	sort.Strings(dateKeys)
	for _, d := range dateKeys {
		row := ts.Rates[d]
		for _, w := range wanted {
			s := stats[w]
			if v, ok := row[w]; ok {
				s.Present++
				if s.FirstSeen == "" {
					s.FirstSeen = d
					s.FirstRate = v
				}
				s.LastSeen = d
				s.LastRate = v
			} else {
				s.Missing++
			}
		}
	}
	fmt.Printf("  per-currency coverage:\n")
	fmt.Printf("    %-4s %-7s %-7s %-12s %-12s %-12s %-12s\n",
		"ccy", "present", "missing", "first", "last", "first_rate", "last_rate")
	for _, w := range wanted {
		s := stats[w]
		fmt.Printf("    %-4s %-7d %-7d %-12s %-12s %-12g %-12g\n",
			s.Currency, s.Present, s.Missing, s.FirstSeen, s.LastSeen, s.FirstRate, s.LastRate)
	}
	fmt.Printf("  fixture         : %s\n", tsPath)
}

func countBusinessDays(start, end time.Time) int {
	n := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		switch d.Weekday() {
		case time.Saturday, time.Sunday:
			continue
		default:
			n++
		}
	}
	return n
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "fatal:", err)
	os.Exit(1)
}
