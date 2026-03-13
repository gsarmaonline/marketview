package deepresearch

import (
	"fmt"
	"log"
	"strings"
)

// Service fetches deep research data for a stock symbol by trying each
// registered provider in order, returning the first successful result.
type Service struct {
	providers []AnnualReportProvider
}

// NewService creates a Service with the given providers tried in order.
func NewService(providers ...AnnualReportProvider) *Service {
	return &Service{providers: providers}
}

// FetchAnnualReports tries each provider in order, returning on the first success.
func (s *Service) FetchAnnualReports(symbol string) ([]AnnualReport, string, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	for _, p := range s.providers {
		reports, err := p.FetchAnnualReports(symbol)
		if err == nil {
			return reports, p.Name(), nil
		}
		log.Printf("annual reports provider %s failed for %s: %v", p.Name(), symbol, err)
	}

	return nil, "", fmt.Errorf("all providers failed for symbol %s", symbol)
}

// Fetch returns the full deep research data for a symbol.
func (s *Service) Fetch(symbol string) (*DeepResearch, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	reports, source, err := s.FetchAnnualReports(symbol)
	if err != nil {
		return nil, err
	}

	return &DeepResearch{
		Symbol:              symbol,
		AnnualReports:       reports,
		AnnualReportsSource: source,
	}, nil
}
