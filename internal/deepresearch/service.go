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
	providers []AnnualReportProvider
	cache     *Cache
}

// NewService creates a Service with the given providers tried in order.
// Pass a non-nil Cache to enable Postgres-backed supply chain caching.
func NewService(cache *Cache, providers ...AnnualReportProvider) *Service {
	return &Service{providers: providers, cache: cache}
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
// are served from the Postgres cache when available; otherwise the most recent
// annual report PDF is parsed and the result is cached for future requests.
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

	// Find the most recent report with a PDF link and populate supply chain.
	for _, r := range reports {
		if r.PDFLink == "" {
			continue
		}

		// Cache hit: return immediately without invoking Python.
		if s.cache != nil {
			if entities, ok, cacheErr := s.cache.Get(ctx, symbol, r.Year); cacheErr == nil && ok {
				dr.SupplyChain = entities
				dr.ParsedReportYear = r.Year
				return dr, nil
			}
		}

		// Cache miss: parse the PDF.
		entities, parseErr := ParseAnnualReport(r.PDFLink)
		if parseErr != nil {
			log.Printf("supply chain parse failed for %s (%s): %v", symbol, r.Year, parseErr)
			break
		}
		dr.SupplyChain = entities
		dr.ParsedReportYear = r.Year

		// Persist for future requests.
		if s.cache != nil {
			if cacheErr := s.cache.Set(ctx, symbol, r.Year, entities); cacheErr != nil {
				log.Printf("supply chain cache write failed for %s (%s): %v", symbol, r.Year, cacheErr)
			}
		}
		break
	}

	return dr, nil
}
