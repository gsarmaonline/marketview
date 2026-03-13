package deepresearch

// AnnualReport represents a single annual report filing from NSE.
type AnnualReport struct {
	SeqNumber int    `json:"seqNumber"`
	Issuer    string `json:"issuer"`
	Year      string `json:"year"`
	Subject   string `json:"subject"`
	PDFLink   string `json:"pdfLink"`
}

// DeepResearch aggregates all deep research data for a stock.
type DeepResearch struct {
	Symbol               string         `json:"symbol"`
	AnnualReports        []AnnualReport `json:"annualReports"`
	AnnualReportsSource  string         `json:"annualReportsSource"` // "NSE" or "BSE"
}
