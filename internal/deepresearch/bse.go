package deepresearch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	bseSearchURL       = "https://api.bseindia.com/BseIndiaAPI/api/GetScripHeaderData/w?Debtflag=&scripcode=&scname=%s&segmentid=0&status=Active"
	bseAnnualReportURL = "https://api.bseindia.com/BseIndiaAPI/api/AnnualReport/w?scripcode=%s"
	bseUserAgent       = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	bseReferer         = "https://www.bseindia.com/"
)

// bseSearchResult is the BSE API response for a company search.
type bseSearchResult struct {
	Table []struct {
		ScripCode string `json:"SECURITY_CODE"`
		ScripName string `json:"SECURITY_NAME"`
	} `json:"Table"`
}

// bseReportItem is a single annual report entry from the BSE API.
type bseReportItem struct {
	ReportYear string `json:"REPORT_YEAR"`
	PDFName    string `json:"PDF_NAME"`
	PDFLink    string `json:"PDF_LINK"`
}

// BSEProvider fetches annual reports from BSE India.
type BSEProvider struct {
	http *http.Client
}

// NewBSEProvider creates a BSEProvider.
func NewBSEProvider() *BSEProvider {
	return &BSEProvider{http: &http.Client{}}
}

func (p *BSEProvider) Name() string { return "BSE" }

func (p *BSEProvider) FetchAnnualReports(symbol string) ([]AnnualReport, error) {
	scripCode, err := p.lookupScripCode(symbol)
	if err != nil {
		return nil, err
	}
	return p.fetchReports(scripCode)
}

func (p *BSEProvider) get(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", bseUserAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", bseReferer)
	req.Header.Set("Origin", "https://www.bseindia.com")

	resp, err := p.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BSE returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (p *BSEProvider) lookupScripCode(symbol string) (string, error) {
	searchURL := fmt.Sprintf(bseSearchURL, url.QueryEscape(symbol))
	body, err := p.get(searchURL)
	if err != nil {
		return "", fmt.Errorf("BSE scrip code lookup: %w", err)
	}

	var result bseSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parsing BSE search response: %w", err)
	}

	if len(result.Table) == 0 {
		return "", fmt.Errorf("no BSE scrip found for symbol %q", symbol)
	}

	return result.Table[0].ScripCode, nil
}

func (p *BSEProvider) fetchReports(scripCode string) ([]AnnualReport, error) {
	reportURL := fmt.Sprintf(bseAnnualReportURL, scripCode)
	body, err := p.get(reportURL)
	if err != nil {
		return nil, fmt.Errorf("BSE annual reports: %w", err)
	}

	var items []bseReportItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("parsing BSE annual reports: %w", err)
	}

	reports := make([]AnnualReport, 0, len(items))
	for i, item := range items {
		reports = append(reports, AnnualReport{
			SeqNumber: i + 1,
			Year:      item.ReportYear,
			Subject:   item.PDFName,
			PDFLink:   item.PDFLink,
		})
	}
	return reports, nil
}
