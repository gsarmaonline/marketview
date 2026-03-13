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

// DeepResearch aggregates all deep research data for a stock.
type DeepResearch struct {
	Symbol              string              `json:"symbol"`
	AnnualReports       []AnnualReport      `json:"annualReports"`
	AnnualReportsSource string              `json:"annualReportsSource"` // "NSE" or "BSE"
	SupplyChain         []SupplyChainEntity `json:"supplyChain"`
	ParsedReportYear    string              `json:"parsedReportYear,omitempty"`
}
