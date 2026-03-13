package deepresearch

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

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

type nseAnnualReportItem struct {
	CompanyName string `json:"companyName"`
	FromYr      string `json:"fromYr"`
	ToYr        string `json:"toYr"`
	FileName    string `json:"fileName"`
}

type nseAnnualReportsResponse struct {
	Data []nseAnnualReportItem `json:"data"`
}

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

	var raw nseAnnualReportsResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing NSE response: %w", err)
	}

	reports := make([]AnnualReport, 0, len(raw.Data))
	for i, item := range raw.Data {
		year := item.FromYr + "-" + item.ToYr
		// Skip zip files — the PDF parser only handles PDFs.
		pdfLink := item.FileName
		if strings.HasSuffix(strings.ToLower(pdfLink), ".zip") {
			pdfLink = ""
		}
		reports = append(reports, AnnualReport{
			SeqNumber: i + 1,
			Issuer:    item.CompanyName,
			Year:      year,
			Subject:   "Annual Report " + year,
			PDFLink:   pdfLink,
		})
	}

	return reports, nil
}
