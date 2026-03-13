package backtest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestRouter() *gin.Engine {
	r := gin.New()
	r.POST("/api/backtest", NewHandler().Handle)
	return r
}

func backtestRequest(symbol, from, to string, capital float64, strategy string) *http.Request {
	body := map[string]interface{}{
		"symbol":  symbol,
		"from":    from,
		"to":      to,
		"capital": capital,
		"strategy": map[string]string{"name": strategy},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/backtest", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// yahooFixtureServer starts an httptest.Server that returns a mock Yahoo Finance chart response.
func yahooFixtureServer(t *testing.T, closes []float64) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
		timestamps := make([]int64, len(closes))
		for i := range closes {
			timestamps[i] = base + int64(i)*86400
		}
		resp := map[string]interface{}{
			"chart": map[string]interface{}{
				"result": []interface{}{
					map[string]interface{}{
						"timestamp": timestamps,
						"indicators": map[string]interface{}{
							"quote": []interface{}{
								map[string]interface{}{
									"open":   closes,
									"high":   closes,
									"low":    closes,
									"close":  closes,
									"volume": make([]int64, len(closes)),
								},
							},
						},
					},
				},
				"error": nil,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	originalURL := yahooChartURL
	yahooChartURL = srv.URL
	t.Cleanup(func() {
		srv.Close()
		yahooChartURL = originalURL
	})
	return srv
}

func TestHandleBacktest_MissingSymbol(t *testing.T) {
	r := newTestRouter()
	body := `{"from":"2020-01-01","to":"2021-01-01","capital":100000,"strategy":{"name":"buy_and_hold"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/backtest", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleBacktest_UnknownStrategy(t *testing.T) {
	r := newTestRouter()
	req := backtestRequest("RELIANCE", "2020-01-01", "2021-01-01", 100000, "does_not_exist")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown strategy, got %d", w.Code)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body) //nolint:errcheck
	if body["error"] == "" {
		t.Error("expected non-empty error field")
	}
}

func TestHandleBacktest_InvalidFromDate(t *testing.T) {
	r := newTestRouter()
	body := `{"symbol":"RELIANCE","from":"not-a-date","to":"2021-01-01","capital":100000,"strategy":{"name":"buy_and_hold"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/backtest", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid from date, got %d", w.Code)
	}
}

func TestHandleBacktest_ToBeforeFrom(t *testing.T) {
	r := newTestRouter()
	req := backtestRequest("RELIANCE", "2021-01-01", "2020-01-01", 100000, "buy_and_hold")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when to < from, got %d", w.Code)
	}
}

func TestHandleBacktest_Success(t *testing.T) {
	yahooFixtureServer(t, []float64{100, 110, 120, 130, 150})

	r := newTestRouter()
	req := backtestRequest("RELIANCE", "2020-01-01", "2020-01-10", 100000, "buy_and_hold")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result Result
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Strategy != "buy_and_hold" {
		t.Errorf("strategy = %q, want buy_and_hold", result.Strategy)
	}
	if result.Capital != 100000 {
		t.Errorf("capital = %f, want 100000", result.Capital)
	}
	if result.FinalValue <= result.Capital {
		t.Errorf("expected profit: final %f should exceed capital %f", result.FinalValue, result.Capital)
	}
	if len(result.Trades) != 1 {
		t.Errorf("expected 1 trade, got %d", len(result.Trades))
	}
	if len(result.EquityCurve) == 0 {
		t.Error("expected non-empty equity curve")
	}
	if result.Metrics.TotalReturnPct <= 0 {
		t.Errorf("expected positive return, got %f%%", result.Metrics.TotalReturnPct)
	}
}

func TestHandleBacktest_SymbolUppercased(t *testing.T) {
	yahooFixtureServer(t, []float64{100, 120})

	r := newTestRouter()
	req := backtestRequest("reliance", "2020-01-01", "2020-01-05", 10000, "buy_and_hold")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for lowercase symbol, got %d: %s", w.Code, w.Body.String())
	}

	var result Result
	json.NewDecoder(w.Body).Decode(&result) //nolint:errcheck
	if result.Symbol != "RELIANCE" {
		t.Errorf("symbol = %q, want RELIANCE", result.Symbol)
	}
}
