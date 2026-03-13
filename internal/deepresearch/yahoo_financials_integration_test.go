package deepresearch

import (
	"testing"
)

// TestFetchYahooFinancials_Integration calls the live Yahoo Finance API for a
// handful of well-known Indian stocks and checks that key fields are populated.
// Run with: go test ./internal/deepresearch/ -run Integration -v
func TestFetchYahooFinancials_Integration(t *testing.T) {
	stocks := []struct {
		symbol string
	}{
		{"RELIANCE"},
		{"TCS"},
		{"INFY"},
		{"HDFCBANK"},
		{"WIPRO"},
	}

	for _, tc := range stocks {
		tc := tc
		t.Run(tc.symbol, func(t *testing.T) {
			f, err := FetchYahooFinancials(tc.symbol)
			if err != nil {
				t.Fatalf("FetchYahooFinancials(%q) error: %v", tc.symbol, err)
			}

			t.Logf("=== %s ===", tc.symbol)
			t.Logf("  P&L:          revenue=%-16s PAT=%-16s PBT=%s",
				f.PnL.RevenueFromOperations, f.PnL.ProfitAfterTax, f.PnL.ProfitBeforeTax)
			t.Logf("  Balance Sheet: equity=%-16s cash=%-16s longTermDebt=%s",
				f.BalanceSheet.TotalEquity, f.BalanceSheet.Cash, f.BalanceSheet.LongTermDebt)
			t.Logf("  Cash Flow:     fromOps=%-16s freeCashFlow=%s",
				f.CashFlow.FromOperations, f.CashFlow.NetChange)
			t.Logf("  Highlights:    EPS=%-8s ROE=%-8s bookValue=%-8s debtToEquity=%s",
				f.Highlights.EPS, f.Highlights.ROE, f.Highlights.BookValuePerShare, f.Highlights.DebtToEquity)

			if f.PnL.RevenueFromOperations == "" {
				t.Errorf("%s: P&L revenueFromOperations is empty", tc.symbol)
			}
			if f.PnL.ProfitAfterTax == "" {
				t.Errorf("%s: P&L profitAfterTax is empty", tc.symbol)
			}
			if f.BalanceSheet.TotalEquity == "" {
				t.Errorf("%s: BalanceSheet totalEquity is empty", tc.symbol)
			}
			if f.BalanceSheet.Cash == "" {
				t.Errorf("%s: BalanceSheet cash is empty", tc.symbol)
			}
			if f.CashFlow.FromOperations == "" {
				// Some Indian stocks (banks, conglomerates) do not have operatingCashflow
				// in Yahoo's financialData; treat as a warning, not a hard failure.
				t.Logf("%s: CashFlow fromOperations is empty (may be unavailable for this stock type)", tc.symbol)
			}
			if f.Highlights.EPS == "" {
				t.Errorf("%s: Highlights EPS is empty", tc.symbol)
			}
		})
	}
}
