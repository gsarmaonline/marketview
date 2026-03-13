package backtest

import (
	"math"
	"testing"
	"time"
)

// makeOHLCV builds a slice of daily OHLCV entries from close prices.
func makeOHLCV(closes []float64, start time.Time) []OHLCV {
	prices := make([]OHLCV, len(closes))
	for i, c := range closes {
		prices[i] = OHLCV{
			Date:  start.AddDate(0, 0, i),
			Open:  c,
			High:  c,
			Low:   c,
			Close: c,
		}
	}
	return prices
}

func TestRun_Profit(t *testing.T) {
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := makeOHLCV([]float64{100, 110, 105, 130}, start)

	result := Run("TEST", prices, BuyAndHold{}, 1000)

	// floor(1000/100) = 10 shares; exit at 130 → final = 1300
	if result.FinalValue != 1300 {
		t.Errorf("final value = %f, want 1300", result.FinalValue)
	}
	if len(result.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(result.Trades))
	}
	tr := result.Trades[0]
	if tr.EntryPrice != 100 {
		t.Errorf("entry price = %f, want 100", tr.EntryPrice)
	}
	if tr.ExitPrice != 130 {
		t.Errorf("exit price = %f, want 130", tr.ExitPrice)
	}
	if tr.Shares != 10 {
		t.Errorf("shares = %f, want 10", tr.Shares)
	}
	if tr.PnL != 300 {
		t.Errorf("pnl = %f, want 300", tr.PnL)
	}
}

func TestRun_Loss(t *testing.T) {
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := makeOHLCV([]float64{200, 150, 100}, start)

	result := Run("TEST", prices, BuyAndHold{}, 1000)

	// floor(1000/200) = 5 shares; exit at 100 → final = 500
	if result.FinalValue != 500 {
		t.Errorf("final value = %f, want 500", result.FinalValue)
	}
	if result.Metrics.TotalReturnPct >= 0 {
		t.Errorf("expected negative return, got %f%%", result.Metrics.TotalReturnPct)
	}
}

func TestRun_CapitalRemainder(t *testing.T) {
	// Capital 150, price 100 → buy 1 share, 50 cash leftover
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := makeOHLCV([]float64{100, 120}, start)

	result := Run("TEST", prices, BuyAndHold{}, 150)

	// 1 share at 120 + 50 cash = 170
	if result.FinalValue != 170 {
		t.Errorf("final value = %f, want 170", result.FinalValue)
	}
}

func TestRun_EmptyPrices(t *testing.T) {
	result := Run("TEST", nil, BuyAndHold{}, 1000)
	if len(result.Trades) != 0 {
		t.Errorf("expected 0 trades for empty prices, got %d", len(result.Trades))
	}
	if len(result.EquityCurve) != 0 {
		t.Errorf("expected empty equity curve, got %d points", len(result.EquityCurve))
	}
}

func TestRun_EquityCurveLength(t *testing.T) {
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := makeOHLCV([]float64{100, 110, 105, 120}, start)

	result := Run("TEST", prices, BuyAndHold{}, 1000)

	if len(result.EquityCurve) != len(prices) {
		t.Errorf("equity curve len = %d, want %d", len(result.EquityCurve), len(prices))
	}
}

func TestRun_EquityCurveDailyValues(t *testing.T) {
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := makeOHLCV([]float64{100, 110, 105, 120}, start)

	result := Run("TEST", prices, BuyAndHold{}, 1000)

	// 10 shares: daily portfolio value = 10 * close
	expected := []float64{1000, 1100, 1050, 1200}
	for i, pt := range result.EquityCurve {
		if pt.Value != expected[i] {
			t.Errorf("equity curve[%d] = %f, want %f", i, pt.Value, expected[i])
		}
	}
}

func TestRun_ResultFields(t *testing.T) {
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := makeOHLCV([]float64{100, 150}, start)

	result := Run("SYM", prices, BuyAndHold{}, 5000)

	if result.Strategy != "buy_and_hold" {
		t.Errorf("strategy = %q, want buy_and_hold", result.Strategy)
	}
	if result.Symbol != "SYM" {
		t.Errorf("symbol = %q, want SYM", result.Symbol)
	}
	if result.Capital != 5000 {
		t.Errorf("capital = %f, want 5000", result.Capital)
	}
	if result.From == "" || result.To == "" {
		t.Error("expected non-empty from/to dates in result")
	}
}

func TestMetrics_TotalReturn(t *testing.T) {
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := makeOHLCV([]float64{100, 150}, start)

	result := Run("TEST", prices, BuyAndHold{}, 1000)

	if math.Abs(result.Metrics.TotalReturnPct-50.0) > 0.01 {
		t.Errorf("total return = %f%%, want 50%%", result.Metrics.TotalReturnPct)
	}
}

func TestMetrics_WinRate(t *testing.T) {
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := makeOHLCV([]float64{100, 150}, start)

	result := Run("TEST", prices, BuyAndHold{}, 1000)

	if result.Metrics.WinRatePct != 100 {
		t.Errorf("win rate = %f%%, want 100%%", result.Metrics.WinRatePct)
	}
	if result.Metrics.TotalTrades != 1 {
		t.Errorf("total trades = %d, want 1", result.Metrics.TotalTrades)
	}
}

func TestMetrics_MaxDrawdown(t *testing.T) {
	// 10 shares: peak=2000 at price 200, trough=1000 at price 100 → 50% drawdown
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := makeOHLCV([]float64{100, 200, 100}, start)

	result := Run("TEST", prices, BuyAndHold{}, 1000)

	if math.Abs(result.Metrics.MaxDrawdownPct-50.0) > 0.01 {
		t.Errorf("max drawdown = %f%%, want 50%%", result.Metrics.MaxDrawdownPct)
	}
}

func TestMetrics_NoDrawdownOnMonotonicallyRising(t *testing.T) {
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := makeOHLCV([]float64{100, 110, 120, 130}, start)

	result := Run("TEST", prices, BuyAndHold{}, 1000)

	if result.Metrics.MaxDrawdownPct != 0 {
		t.Errorf("max drawdown = %f%%, want 0%%", result.Metrics.MaxDrawdownPct)
	}
}

func TestComputeSharpe_InsufficientData(t *testing.T) {
	sharpe := computeSharpe([]EquityPoint{})
	if sharpe != 0 {
		t.Errorf("sharpe for empty curve = %f, want 0", sharpe)
	}
	sharpe = computeSharpe([]EquityPoint{{Date: "2020-01-01", Value: 1000}})
	if sharpe != 0 {
		t.Errorf("sharpe for single point = %f, want 0", sharpe)
	}
}

func TestComputeSharpe_FlatReturns(t *testing.T) {
	// All same values → std dev = 0 → sharpe = 0
	curve := []EquityPoint{
		{Date: "2020-01-01", Value: 1000},
		{Date: "2020-01-02", Value: 1000},
		{Date: "2020-01-03", Value: 1000},
	}
	sharpe := computeSharpe(curve)
	if sharpe != 0 {
		t.Errorf("sharpe for flat returns = %f, want 0", sharpe)
	}
}
