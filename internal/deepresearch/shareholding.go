package deepresearch

import (
	"encoding/json"
	"fmt"
	"io"

	"marketview/internal/nse"
)

const nseShareholdingURL = "https://www.nseindia.com/api/corporate-share-holdings-new?index=equities&symbol=%s"

// nseShareholdingFiling is one quarterly entry from the NSE list response.
type nseShareholdingFiling struct {
	Quarter                  string `json:"quarter"`
	PromoterAndPromoterGroup string `json:"promoterAndPromoterGroup"`
	PublicShareholding       string `json:"publicShareholding"`
	Fii                      string `json:"fii"`
	Dii                      string `json:"dii"`
	MutualFunds              string `json:"mutualFunds"`
}

type nseShareholdingListResponse struct {
	Data []nseShareholdingFiling `json:"data"`
}

// nseShareholdingDetailResponse is the per-filing detailed breakdown,
// including top shareholders. Used with the detail endpoint when available.
type nseShareholdingDetailEntry struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Shares   string `json:"shares"`
	Percent  string `json:"percentage"`
}

// FetchNSEShareholding fetches the most recent quarterly shareholding pattern
// for symbol using the shared NSE session client.
func FetchNSEShareholding(client *nse.Client, symbol string) (*ShareholdingPattern, error) {
	url := fmt.Sprintf(nseShareholdingURL, symbol)

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("NSE shareholding request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading NSE shareholding response: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("NSE shareholding returned status %d", resp.StatusCode)
	}

	var raw nseShareholdingListResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing NSE shareholding response: %w", err)
	}
	if len(raw.Data) == 0 {
		return nil, fmt.Errorf("no shareholding data returned for %s", symbol)
	}

	latest := raw.Data[0]
	sp := &ShareholdingPattern{
		QuarterEndDate: latest.Quarter,
		Category: ShareholdingCategory{
			PromoterAndPromoterGroup: latest.PromoterAndPromoterGroup,
			FII:                      latest.Fii,
			DII:                      latest.Dii,
			MutualFunds:              latest.MutualFunds,
			PublicAndOthers:          latest.PublicShareholding,
		},
	}

	return sp, nil
}
