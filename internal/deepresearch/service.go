package deepresearch

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"marketview/internal/nse"
)

const annualReportsURL = "https://www.nseindia.com/api/annual-reports?index=equities&symbol=%s&period=annual"

// Service fetches deep research data for a stock symbol.
type Service struct {
	nseClient *nse.Client
	bse       *bseClient
}

// NewService creates a new deep research Service.
func NewService(nseClient *nse.Client) *Service {
	return &Service{
		nseClient: nseClient,
		bse:       newBSEClient(),
	}
}

// fetchFromNSE retrieves annual reports via NSE.
func (s *Service) fetchFromNSE(symbol string) ([]AnnualReport, error) {
	url := fmt.Sprintf(annualReportsURL, symbol)

	resp, err := s.nseClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("NSE request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading NSE response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("NSE returned status %d", resp.StatusCode)
	}

	var reports []AnnualReport
	if err := json.Unmarshal(body, &reports); err != nil {
		return nil, fmt.Errorf("parsing NSE response: %w", err)
	}

	return reports, nil
}

// fetchFromBSE retrieves annual reports via BSE using a symbol lookup.
func (s *Service) fetchFromBSE(symbol string) ([]AnnualReport, error) {
	scripCode, err := s.bse.lookupScripCode(symbol)
	if err != nil {
		return nil, err
	}
	return s.bse.fetchAnnualReports(scripCode)
}

// FetchAnnualReports tries NSE first; falls back to BSE on failure.
func (s *Service) FetchAnnualReports(symbol string) ([]AnnualReport, string, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	reports, err := s.fetchFromNSE(symbol)
	if err == nil {
		return reports, "NSE", nil
	}
	log.Printf("NSE annual reports failed for %s (%v), trying BSE", symbol, err)

	reports, err = s.fetchFromBSE(symbol)
	if err != nil {
		return nil, "", fmt.Errorf("both NSE and BSE failed: %w", err)
	}
	return reports, "BSE", nil
}

// Fetch returns the full deep research data for a symbol.
func (s *Service) Fetch(symbol string) (*DeepResearch, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	reports, source, err := s.FetchAnnualReports(symbol)
	if err != nil {
		return nil, err
	}

	return &DeepResearch{
		Symbol:               symbol,
		AnnualReports:        reports,
		AnnualReportsSource: source,
	}, nil
}
