package deepresearch

// AnnualReportProvider is the interface that any annual report data source must implement.
// Providers are tried in the order they are registered with the Service; the first
// successful response wins.
type AnnualReportProvider interface {
	// Name returns the human-readable identifier for this provider (e.g. "NSE", "BSE").
	Name() string
	// FetchAnnualReports returns the list of annual reports for the given NSE symbol.
	FetchAnnualReports(symbol string) ([]AnnualReport, error)
}
