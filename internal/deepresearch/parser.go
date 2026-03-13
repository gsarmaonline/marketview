package deepresearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// parserURL returns the base URL of the Python PDF parser service.
// Defaults to http://localhost:5001; override with PARSER_URL env var.
func parserURL() string {
	if u := os.Getenv("PARSER_URL"); u != "" {
		return u
	}
	return "http://localhost:5001"
}

var parserHTTP = &http.Client{Timeout: 120 * time.Second}

type parseRequest struct {
	URL string `json:"url"`
}

type parseResponse struct {
	Companies  []SupplyChainEntity `json:"companies"`
	Financials *Financials         `json:"financials,omitempty"`
	Error      string              `json:"error,omitempty"`
}

// ParseAnnualReport calls the Python parser service to extract supply chain
// entities and financial data from the given PDF.
func ParseAnnualReport(pdfURL string) ([]SupplyChainEntity, *Financials, error) {
	body, err := json.Marshal(parseRequest{URL: pdfURL})
	if err != nil {
		return nil, nil, fmt.Errorf("pdf parser: marshal request: %w", err)
	}

	resp, err := parserHTTP.Post(parserURL()+"/parse", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("pdf parser: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("pdf parser: read response: %w", err)
	}

	var result parseResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, nil, fmt.Errorf("pdf parser: decode response: %w", err)
	}
	if result.Error != "" {
		return nil, nil, fmt.Errorf("pdf parser: %s", result.Error)
	}
	return result.Companies, result.Financials, nil
}
