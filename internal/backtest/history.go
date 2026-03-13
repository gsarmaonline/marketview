package backtest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var yahooChartURL = "https://query1.finance.yahoo.com/v8/finance/chart"

type yahooChartResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []float64 `json:"open"`
					High   []float64 `json:"high"`
					Low    []float64 `json:"low"`
					Close  []float64 `json:"close"`
					Volume []int64   `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

// FetchHistory retrieves daily OHLCV data for a symbol between from and to.
// Appends ".NS" if no exchange suffix is present (NSE stocks).
func FetchHistory(symbol string, from, to time.Time) ([]OHLCV, error) {
	if !strings.Contains(symbol, ".") {
		symbol = symbol + ".NS"
	}

	url := fmt.Sprintf("%s/%s?interval=1d&period1=%d&period2=%d",
		yahooChartURL, symbol, from.Unix(), to.Unix())

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo chart fetch: %w", err)
	}
	defer resp.Body.Close()

	var result yahooChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("yahoo chart decode: %w", err)
	}

	if len(result.Chart.Result) == 0 {
		return nil, fmt.Errorf("no historical data for %q", symbol)
	}

	r := result.Chart.Result[0]
	if len(r.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no quote data for %q", symbol)
	}

	q := r.Indicators.Quote[0]
	prices := make([]OHLCV, 0, len(r.Timestamp))
	for i, ts := range r.Timestamp {
		if i >= len(q.Close) || q.Close[i] == 0 {
			continue // skip missing/invalid data points
		}
		prices = append(prices, OHLCV{
			Date:   time.Unix(ts, 0).UTC(),
			Open:   sliceGet(q.Open, i),
			High:   sliceGet(q.High, i),
			Low:    sliceGet(q.Low, i),
			Close:  q.Close[i],
			Volume: sliceGetInt(q.Volume, i),
		})
	}

	return prices, nil
}

func sliceGet(s []float64, i int) float64 {
	if i < len(s) {
		return s[i]
	}
	return 0
}

func sliceGetInt(s []int64, i int) int64 {
	if i < len(s) {
		return s[i]
	}
	return 0
}
