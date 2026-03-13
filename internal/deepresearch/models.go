package deepresearch

// AnnualReport represents a single annual report filing from NSE.
type AnnualReport struct {
	SeqNumber int    `json:"seqNumber"`
	Issuer    string `json:"issuer"`
	Year      string `json:"year"`
	Subject   string `json:"subject"`
	PDFLink   string `json:"pdfLink"`
}

// SupplyChainEntity is a company extracted from the Related Party Transactions
// section of an annual report.
type SupplyChainEntity struct {
	Name         string `json:"name"`
	Relationship string `json:"relationship"` // e.g. "subsidiary", "supplier", "customer"
	Amount       string `json:"amount,omitempty"`
}

// ProfitAndLoss holds key P&L statement line items.
type ProfitAndLoss struct {
	RevenueFromOperations string `json:"revenueFromOperations,omitempty"`
	OtherIncome           string `json:"otherIncome,omitempty"`
	TotalIncome           string `json:"totalIncome,omitempty"`
	MaterialCost          string `json:"materialCost,omitempty"`
	EmployeeBenefits      string `json:"employeeBenefits,omitempty"`
	FinanceCosts          string `json:"financeCosts,omitempty"`
	Depreciation          string `json:"depreciation,omitempty"`
	OtherExpenses         string `json:"otherExpenses,omitempty"`
	TotalExpenses         string `json:"totalExpenses,omitempty"`
	ProfitBeforeTax       string `json:"profitBeforeTax,omitempty"`
	TaxExpense            string `json:"taxExpense,omitempty"`
	ProfitAfterTax        string `json:"profitAfterTax,omitempty"`
}

// BalanceSheet holds key balance sheet line items.
type BalanceSheet struct {
	TotalAssets        string `json:"totalAssets,omitempty"`
	FixedAssets        string `json:"fixedAssets,omitempty"`
	CurrentAssets      string `json:"currentAssets,omitempty"`
	Cash               string `json:"cash,omitempty"`
	Inventory          string `json:"inventory,omitempty"`
	Receivables        string `json:"receivables,omitempty"`
	TotalEquity        string `json:"totalEquity,omitempty"`
	LongTermDebt       string `json:"longTermDebt,omitempty"`
	CurrentLiabilities string `json:"currentLiabilities,omitempty"`
}

// CashFlow holds key cash flow statement line items.
type CashFlow struct {
	FromOperations string `json:"fromOperations,omitempty"`
	FromInvesting  string `json:"fromInvesting,omitempty"`
	FromFinancing  string `json:"fromFinancing,omitempty"`
	NetChange      string `json:"netChange,omitempty"`
}

// FinancialHighlights holds key per-share and ratio metrics.
type FinancialHighlights struct {
	EPS               string `json:"eps,omitempty"`
	BookValuePerShare string `json:"bookValuePerShare,omitempty"`
	DividendPerShare  string `json:"dividendPerShare,omitempty"`
	ROE               string `json:"roe,omitempty"`
	ROCE              string `json:"roce,omitempty"`
	DebtToEquity      string `json:"debtToEquity,omitempty"`
}

// Financials bundles all financial statement data for a stock.
type Financials struct {
	PnL          ProfitAndLoss       `json:"pnl"`
	BalanceSheet BalanceSheet        `json:"balanceSheet"`
	CashFlow     CashFlow            `json:"cashFlow"`
	Highlights   FinancialHighlights `json:"highlights"`
}

// ShareholderEntry represents a named entity in the top shareholders list.
type ShareholderEntry struct {
	Name           string `json:"name"`
	NoOfShares     string `json:"noOfShares,omitempty"`
	PercentageHeld string `json:"percentageHeld"`
}

// ShareholdingCategory breaks down aggregate percentages by investor class.
type ShareholdingCategory struct {
	PromoterAndPromoterGroup string `json:"promoterAndPromoterGroup"`
	FII                      string `json:"fii,omitempty"`
	DII                      string `json:"dii,omitempty"`
	MutualFunds              string `json:"mutualFunds,omitempty"`
	PublicAndOthers          string `json:"publicAndOthers"`
}

// ShareholdingPattern holds the most recent quarterly shareholding breakdown.
type ShareholdingPattern struct {
	QuarterEndDate  string               `json:"quarterEndDate"`
	Category        ShareholdingCategory `json:"category"`
	TopShareholders []ShareholderEntry   `json:"topShareholders,omitempty"`
}

// DeepResearch aggregates all deep research data for a stock.
type DeepResearch struct {
	Symbol              string               `json:"symbol"`
	AnnualReports       []AnnualReport       `json:"annualReports"`
	AnnualReportsSource string               `json:"annualReportsSource"` // "NSE" or "BSE"
	SupplyChain         []SupplyChainEntity  `json:"supplyChain"`
	Financials          *Financials          `json:"financials,omitempty"`
	ShareholdingPattern *ShareholdingPattern `json:"shareholdingPattern,omitempty"`
	ParsedReportYear    string               `json:"parsedReportYear,omitempty"`
}
