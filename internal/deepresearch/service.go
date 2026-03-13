package deepresearch

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// Service fetches deep research data for a stock symbol by trying each
// registered provider in order, returning the first successful result.
type Service struct {
	providers         []AnnualReportProvider
	store             StoreInterface
	financialsFetcher func(symbol string) (*Financials, error)
}

// NewService creates a Service with the given providers tried in order.
// Pass a non-nil StoreInterface to enable Postgres-backed supply chain persistence.
func NewService(store StoreInterface, providers ...AnnualReportProvider) *Service {
	return &Service{
		providers:         providers,
		store:             store,
		financialsFetcher: FetchYahooFinancials,
	}
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

// Fetch returns the full deep research data for a symbol. Supply chain entities
// are served from the store when available; otherwise the most recent annual
// report PDF is parsed and the result is saved for future requests. Financials
// are fetched live from Yahoo Finance.
func (s *Service) Fetch(ctx context.Context, symbol string) (*DeepResearch, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	reports, source, err := s.FetchAnnualReports(symbol)
	if err != nil {
		return nil, err
	}

	dr := &DeepResearch{
		Symbol:              symbol,
		AnnualReports:       reports,
		AnnualReportsSource: source,
	}

	// Fetch financials from Yahoo Finance.
	if s.financialsFetcher != nil {
		if f, fErr := s.financialsFetcher(symbol); fErr == nil {
			dr.Financials = f
		} else {
			log.Printf("yahoo financials failed for %s: %v", symbol, fErr)
		}
	}

	// Find the most recent report with a PDF link and populate supply chain.
	for _, r := range reports {
		if r.PDFLink == "" {
			continue
		}

		// Store hit: return cached supply chain without invoking the parser.
		if s.store != nil {
			if entities, ok, storeErr := s.store.Get(ctx, symbol, r.Year); storeErr == nil && ok {
				dr.SupplyChain = entities
				dr.ParsedReportYear = r.Year
				return dr, nil
			}
		}

		// Not yet parsed: call the PDF parser service.
		entities, parseErr := ParseAnnualReport(r.PDFLink)
		if parseErr != nil {
			log.Printf("supply chain parse failed for %s (%s): %v", symbol, r.Year, parseErr)
			break
		}
		dr.SupplyChain = entities
		dr.ParsedReportYear = r.Year

		// Persist for future requests.
		if s.store != nil {
			if storeErr := s.store.Set(ctx, symbol, r.Year, entities); storeErr != nil {
				log.Printf("supply chain store write failed for %s (%s): %v", symbol, r.Year, storeErr)
			}
		}
		break
	}

	return dr, nil
}
