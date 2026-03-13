package deepresearch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"marketview/internal/yahoo"
)

const yahooQuoteSummaryURL = "https://query1.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=%s&crumb=%s"

type yahooRawValue struct {
	Raw float64 `json:"raw"`
}

type yahooIncomeStatementEntry struct {
	TotalRevenue                yahooRawValue `json:"totalRevenue"`
	TotalOperatingExpenses      yahooRawValue `json:"totalOperatingExpenses"`
	NetIncome                   yahooRawValue `json:"netIncome"`
	IncomeTaxExpense            yahooRawValue `json:"incomeTaxExpense"`
	DepreciationAndAmortization yahooRawValue `json:"depreciationAndAmortization"`
	InterestExpense             yahooRawValue `json:"interestExpense"`
	TotalOtherIncomeExpenseNet  yahooRawValue `json:"totalOtherIncomeExpenseNet"`
	IncomeBeforeTax             yahooRawValue `json:"incomeBeforeTax"`
}

type yahooFinancialsResponse struct {
	QuoteSummary struct {
		Result []struct {
			IncomeStatementHistory struct {
				IncomeStatementHistory []yahooIncomeStatementEntry `json:"incomeStatementHistory"`
			} `json:"incomeStatementHistory"`
			DefaultKeyStatistics struct {
				TrailingEps               yahooRawValue `json:"trailingEps"`
				BookValue                 yahooRawValue `json:"bookValue"`
				ForwardAnnualDividendRate yahooRawValue `json:"forwardAnnualDividendRate"`
				SharesOutstanding         yahooRawValue `json:"sharesOutstanding"`
			} `json:"defaultKeyStatistics"`
			// financialData provides balance sheet and cash flow aggregates available
			// for Indian stocks (Yahoo's balanceSheetHistory/cashflowStatementHistory
			// modules return empty line items for NSE/BSE symbols).
			FinancialData struct {
				ReturnOnEquity  yahooRawValue `json:"returnOnEquity"`
				DebtToEquity    yahooRawValue `json:"debtToEquity"`
				TotalCash       yahooRawValue `json:"totalCash"`
				TotalDebt       yahooRawValue `json:"totalDebt"`
				OperatingCashflow yahooRawValue `json:"operatingCashflow"`
				FreeCashflow    yahooRawValue `json:"freeCashflow"`
			} `json:"financialData"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"quoteSummary"`
}

// rawToStr converts a raw float64 to an integer string. Returns "" for zero.
func rawToStr(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatInt(int64(v), 10)
}

// floatToStr formats a float to 2 decimal places. Returns "" for zero.
func floatToStr(v float64) string {
	if v == 0 {
		return ""
	}
	return fmt.Sprintf("%.2f", v)
}

// pctToStr converts a decimal ratio (e.g. 0.18) to a percentage string "18.00".
func pctToStr(v float64) string {
	if v == 0 {
		return ""
	}
	return fmt.Sprintf("%.2f", v*100)
}

// FetchYahooFinancials retrieves the latest annual financial statements for a
// stock symbol. For Indian stocks it tries SYMBOL.NS (NSE) then SYMBOL.BO (BSE).
func FetchYahooFinancials(symbol string) (*Financials, error) {
	for _, suffix := range []string{".NS", ".BO"} {
		f, err := fetchYahooFinancialsFor(symbol + suffix)
		if err == nil && f != nil {
			return f, nil
		}
	}
	return nil, fmt.Errorf("yahoo financials: no data found for %s", symbol)
}

func fetchYahooFinancialsFor(yahooSymbol string) (*Financials, error) {
	crumb, cookies, err := yahoo.GetCrumb()
	if err != nil {
		return nil, fmt.Errorf("yahoo: could not get crumb: %w", err)
	}

	modules := "incomeStatementHistory,defaultKeyStatistics,financialData"
	url := fmt.Sprintf(yahooQuoteSummaryURL, yahooSymbol, modules, crumb)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo quoteSummary: %w", err)
	}
	defer resp.Body.Close()

	var result yahooFinancialsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("yahoo decode: %w", err)
	}

	if len(result.QuoteSummary.Result) == 0 {
		return nil, fmt.Errorf("yahoo: no result for %s", yahooSymbol)
	}

	r := result.QuoteSummary.Result[0]
	f := &Financials{}

	// P&L — most recent annual entry
	if len(r.IncomeStatementHistory.IncomeStatementHistory) > 0 {
		is := r.IncomeStatementHistory.IncomeStatementHistory[0]
		totalIncome := is.TotalRevenue.Raw + is.TotalOtherIncomeExpenseNet.Raw
		f.PnL = ProfitAndLoss{
			RevenueFromOperations: rawToStr(is.TotalRevenue.Raw),
			OtherIncome:           rawToStr(is.TotalOtherIncomeExpenseNet.Raw),
			TotalIncome:           rawToStr(totalIncome),
			TotalExpenses:         rawToStr(is.TotalOperatingExpenses.Raw),
			ProfitBeforeTax:       rawToStr(is.IncomeBeforeTax.Raw),
			TaxExpense:            rawToStr(is.IncomeTaxExpense.Raw),
			ProfitAfterTax:        rawToStr(is.NetIncome.Raw),
			Depreciation:          rawToStr(is.DepreciationAndAmortization.Raw),
			FinanceCosts:          rawToStr(is.InterestExpense.Raw),
		}
	}

	ks := r.DefaultKeyStatistics
	fd := r.FinancialData

	// Balance Sheet — Yahoo's balanceSheetHistory module returns empty line items
	// for Indian stocks; use financialData aggregates + derive equity from book value.
	totalEquity := ks.BookValue.Raw * ks.SharesOutstanding.Raw
	f.BalanceSheet = BalanceSheet{
		Cash:         rawToStr(fd.TotalCash.Raw),
		TotalEquity:  rawToStr(totalEquity),
		LongTermDebt: rawToStr(fd.TotalDebt.Raw),
	}

	// Cash Flow — operatingCashflow is reliable; investing/financing not available.
	f.CashFlow = CashFlow{
		FromOperations: rawToStr(fd.OperatingCashflow.Raw),
		NetChange:      rawToStr(fd.FreeCashflow.Raw),
	}

	// Highlights
	f.Highlights = FinancialHighlights{
		EPS:               floatToStr(ks.TrailingEps.Raw),
		BookValuePerShare: floatToStr(ks.BookValue.Raw),
		DividendPerShare:  floatToStr(ks.ForwardAnnualDividendRate.Raw),
		ROE:               pctToStr(fd.ReturnOnEquity.Raw),
		DebtToEquity:      floatToStr(fd.DebtToEquity.Raw / 100),
	}

	return f, nil
}
