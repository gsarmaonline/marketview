// Yahoo Finance probe.
//
// Hits query1.finance.yahoo.com/v8/finance/chart for a set of country index
// symbols, saves the raw JSON response as a fixture, and prints a one-line
// summary per symbol so we can answer the questions in PLAN.md before writing
// the production PriceProvider.
//
// The probe is deliberately not a reusable library. Each provider's probe is
// its own script because each API is weird in its own way.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	chartHost     = "https://query1.finance.yahoo.com"
	chartPath     = "/v8/finance/chart/%s"
	crumbURL      = "https://query1.finance.yahoo.com/v1/test/getcrumb"
	cookieSeedURL = "https://fc.yahoo.com"
	userAgent     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

var defaultSymbols = []string{
	"^GSPC",  // S&P 500
	"^IXIC",  // Nasdaq Composite
	"^NSEI",  // Nifty 50
	"^BSESN", // BSE Sensex
	"^N225",  // Nikkei 225
	"^FTSE",  // FTSE 100
	"^GDAXI", // DAX
	"^HSI",   // Hang Seng
	"^BVSP",  // Bovespa
	"^KS11",  // KOSPI
	"^AXJO",  // ASX 200
}

type chartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency             string  `json:"currency"`
				Symbol               string  `json:"symbol"`
				ExchangeName         string  `json:"exchangeName"`
				ExchangeTimezoneName string  `json:"exchangeTimezoneName"`
				RegularMarketPrice   float64 `json:"regularMarketPrice"`
				FirstTradeDate       int64   `json:"firstTradeDate"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []*float64 `json:"open"`
					High   []*float64 `json:"high"`
					Low    []*float64 `json:"low"`
					Close  []*float64 `json:"close"`
					Volume []*int64   `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

type finding struct {
	Symbol         string
	HTTPStatus     int
	APIError       string
	BarsReturned   int
	FirstDate      string
	LastDate       string
	Currency       string
	Timezone       string
	NullCloses     int
	CrumbRequired  bool
	FixturePath    string
	BytesWritten   int
	DurationMillis int64
}

type client struct {
	http  *http.Client
	crumb string
}

func newClient() (*client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &client{
		http: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
	}, nil
}

func (c *client) ensureCrumb(ctx context.Context) error {
	if c.crumb != "" {
		return nil
	}
	// Seed the cookie jar.
	if _, err := c.do(ctx, http.MethodGet, cookieSeedURL, nil); err != nil {
		return fmt.Errorf("seed cookie: %w", err)
	}
	body, err := c.do(ctx, http.MethodGet, crumbURL, nil)
	if err != nil {
		return fmt.Errorf("fetch crumb: %w", err)
	}
	crumb := strings.TrimSpace(string(body))
	if crumb == "" {
		return errors.New("empty crumb")
	}
	c.crumb = crumb
	return nil
}

func (c *client) do(ctx context.Context, method, rawURL string, query url.Values) ([]byte, error) {
	if query != nil {
		rawURL = rawURL + "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json,text/plain,*/*")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return body, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	return body, nil
}

func (c *client) fetchChart(ctx context.Context, symbol string, from, to time.Time) ([]byte, int, bool, error) {
	build := func(includeCrumb bool) (url.Values, error) {
		q := url.Values{}
		q.Set("period1", fmt.Sprintf("%d", from.Unix()))
		q.Set("period2", fmt.Sprintf("%d", to.Unix()))
		q.Set("interval", "1d")
		q.Set("events", "div,split")
		if includeCrumb {
			if err := c.ensureCrumb(ctx); err != nil {
				return nil, err
			}
			q.Set("crumb", c.crumb)
		}
		return q, nil
	}

	endpoint := chartHost + fmt.Sprintf(chartPath, url.PathEscape(symbol))

	// First attempt: no crumb. The chart endpoint historically works without one.
	q, _ := build(false)
	body, err := c.do(ctx, http.MethodGet, endpoint, q)
	if err == nil && !looksLikeAuthError(body) {
		return body, http.StatusOK, false, nil
	}

	// Retry with a crumb if it looks like an auth issue.
	q2, qErr := build(true)
	if qErr != nil {
		return body, statusFromErr(err), false, qErr
	}
	body2, err2 := c.do(ctx, http.MethodGet, endpoint, q2)
	if err2 != nil {
		return body2, statusFromErr(err2), true, err2
	}
	return body2, http.StatusOK, true, nil
}

func looksLikeAuthError(body []byte) bool {
	s := string(body)
	return strings.Contains(s, "Invalid Crumb") || strings.Contains(s, "Unauthorized")
}

func statusFromErr(err error) int {
	if err == nil {
		return http.StatusOK
	}
	// crude: extract "status N" if present
	s := err.Error()
	if i := strings.Index(s, "status "); i >= 0 {
		var n int
		fmt.Sscanf(s[i:], "status %d", &n)
		if n > 0 {
			return n
		}
	}
	return 0
}

func summarise(symbol string, body []byte, status int, crumbUsed bool, fixturePath string, dur time.Duration) finding {
	f := finding{
		Symbol:         symbol,
		HTTPStatus:     status,
		FixturePath:    fixturePath,
		BytesWritten:   len(body),
		CrumbRequired:  crumbUsed,
		DurationMillis: dur.Milliseconds(),
	}
	var parsed chartResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		f.APIError = fmt.Sprintf("parse: %v", err)
		return f
	}
	if parsed.Chart.Error != nil {
		f.APIError = parsed.Chart.Error.Code + ": " + parsed.Chart.Error.Description
		return f
	}
	if len(parsed.Chart.Result) == 0 {
		f.APIError = "empty result"
		return f
	}
	r := parsed.Chart.Result[0]
	f.Currency = r.Meta.Currency
	f.Timezone = r.Meta.ExchangeTimezoneName
	f.BarsReturned = len(r.Timestamp)
	if len(r.Timestamp) > 0 {
		f.FirstDate = time.Unix(r.Timestamp[0], 0).UTC().Format("2006-01-02")
		f.LastDate = time.Unix(r.Timestamp[len(r.Timestamp)-1], 0).UTC().Format("2006-01-02")
	}
	if len(r.Indicators.Quote) > 0 {
		for _, c := range r.Indicators.Quote[0].Close {
			if c == nil {
				f.NullCloses++
			}
		}
	}
	return f
}

func safeName(symbol string) string {
	r := strings.NewReplacer("^", "_caret_", "/", "_slash_", "=", "_eq_")
	return r.Replace(symbol)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func main() {
	symbolsFlag := flag.String("symbols", strings.Join(defaultSymbols, ","), "comma-separated Yahoo symbols")
	years := flag.Int("years", 5, "history depth in years")
	outDir := flag.String("out", "testdata/yahoo", "fixture output directory")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		die(err)
	}

	c, err := newClient()
	if err != nil {
		die(err)
	}

	ctx := context.Background()
	to := time.Now().UTC()
	from := to.AddDate(-*years, 0, 0)
	symbols := strings.Split(*symbolsFlag, ",")

	fmt.Printf("probing %d symbol(s), %d years of daily data, range %s..%s\n",
		len(symbols), *years, from.Format("2006-01-02"), to.Format("2006-01-02"))
	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("%-10s %-4s %-6s %-12s %-12s %-4s %-6s %-6s %-9s  %s\n",
		"symbol", "http", "bars", "first", "last", "ccy", "nulls", "crumb", "ms", "fixture")

	for _, sym := range symbols {
		sym = strings.TrimSpace(sym)
		if sym == "" {
			continue
		}
		start := time.Now()
		body, status, crumbUsed, ferr := c.fetchChart(ctx, sym, from, to)
		dur := time.Since(start)

		fixturePath := filepath.Join(*outDir, fmt.Sprintf("chart_%s.json", safeName(sym)))
		if len(body) > 0 {
			if werr := os.WriteFile(fixturePath, body, 0o644); werr != nil {
				fmt.Fprintln(os.Stderr, "write fixture:", werr)
			}
		}

		f := summarise(sym, body, status, crumbUsed, fixturePath, dur)
		if ferr != nil && f.APIError == "" {
			f.APIError = ferr.Error()
		}

		errCol := f.APIError
		if errCol == "" {
			errCol = fmt.Sprintf("ccy=%s tz=%s", f.Currency, f.Timezone)
		}
		fmt.Printf("%-10s %-4d %-6d %-12s %-12s %-4s %-6d %-6t %-9d  %s\n",
			f.Symbol, f.HTTPStatus, f.BarsReturned, f.FirstDate, f.LastDate,
			f.Currency, f.NullCloses, f.CrumbRequired, f.DurationMillis, errCol)
	}
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "fatal:", err)
	os.Exit(1)
}
