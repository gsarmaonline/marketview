package deepresearch

import (
	"encoding/json"
	"fmt"
	"io"

	"marketview/internal/nse"
)

const nseAnnualReportsURL = "https://www.nseindia.com/api/annual-reports?index=equities&symbol=%s&period=annual"

// NSEProvider fetches annual reports from NSE India.
type NSEProvider struct {
	client *nse.Client
}

// NewNSEProvider creates an NSEProvider using the shared NSE HTTP client.
func NewNSEProvider(client *nse.Client) *NSEProvider {
	return &NSEProvider{client: client}
}

func (p *NSEProvider) Name() string { return "NSE" }

func (p *NSEProvider) FetchAnnualReports(symbol string) ([]AnnualReport, error) {
	url := fmt.Sprintf(nseAnnualReportsURL, symbol)

	resp, err := p.client.Get(url)
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
