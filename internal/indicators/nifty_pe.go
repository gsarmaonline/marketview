package indicators

import (
	"encoding/json"
	"fmt"
	"io"
	"marketview/internal/nse"
)

const niftyPEURL = "https://www.nseindia.com/api/equity-stockIndices?index=NIFTY%2050"

// niftyPEResponse is the relevant subset of the NSE API response.
type niftyPEResponse struct {
	Data []struct {
		Symbol string  `json:"symbol"`
		PE     float64 `json:"pe"`
		PB     float64 `json:"pb"`
	} `json:"data"`
}

// NiftyPE fetches the NIFTY 50 Price-to-Earnings ratio from NSE.
type NiftyPE struct {
	client *nse.Client
}

func NewNiftyPE(client *nse.Client) *NiftyPE {
	return &NiftyPE{client: client}
}

func (n *NiftyPE) Name() string {
	return "NIFTY 50 PE Ratio"
}

func (n *NiftyPE) Fetch() (IndicatorResult, error) {
	resp, err := n.client.Get(niftyPEURL)
	if err != nil {
		return IndicatorResult{}, fmt.Errorf("fetching NIFTY PE: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return IndicatorResult{}, fmt.Errorf("reading NIFTY PE response: %w", err)
	}

	var data niftyPEResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return IndicatorResult{}, fmt.Errorf("parsing NIFTY PE response: %w", err)
	}

	pe, err := extractNiftyPE(data)
	if err != nil {
		return IndicatorResult{}, err
	}

	signal := scorePE(pe)

	return IndicatorResult{
		Name:        n.Name(),
		Value:       pe,
		Unit:        "x",
		Signal:      signal.String(),
		Description: peDescription(pe, signal),
	}, nil
}

// extractNiftyPE finds the NIFTY 50 entry in the response data.
func extractNiftyPE(data niftyPEResponse) (float64, error) {
	for _, d := range data.Data {
		if d.Symbol == "NIFTY 50" {
			return d.PE, nil
		}
	}
	return 0, fmt.Errorf("NIFTY 50 entry not found in NSE response")
}

// scorePE applies the valuation thresholds for NIFTY PE.
//
//	< 20  → Bullish  (historically cheap)
//	20–25 → Neutral  (fair value range)
//	> 25  → Bearish  (expensive)
func scorePE(pe float64) Signal {
	switch {
	case pe < 20:
		return Bullish
	case pe > 25:
		return Bearish
	default:
		return Neutral
	}
}

func peDescription(pe float64, signal Signal) string {
	switch signal {
	case Bullish:
		return fmt.Sprintf("PE of %.1fx is below 20 — market is historically cheap", pe)
	case Bearish:
		return fmt.Sprintf("PE of %.1fx is above 25 — market is historically expensive", pe)
	default:
		return fmt.Sprintf("PE of %.1fx is in the fair-value range (20–25)", pe)
	}
}
