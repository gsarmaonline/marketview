package stock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"marketview/internal/yahoo"
)

const yahooSummaryURL = "https://query1.finance.yahoo.com/v10/finance/quoteSummary"

type yahooValue struct {
	Raw float64 `json:"raw"`
}

type yahooPriceResponse struct {
	QuoteSummary struct {
		Result []struct {
			Price struct {
				RegularMarketPrice yahooValue `json:"regularMarketPrice"`
				Currency           string     `json:"currency"`
				ShortName          string     `json:"shortName"`
			} `json:"price"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"quoteSummary"`
}

// PriceResult holds the fetched price for a stock symbol.
type PriceResult struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Currency  string  `json:"currency"`
	ShortName string  `json:"short_name"`
}

// FetchPrice fetches the current market price for an NSE stock symbol via Yahoo Finance.
// It appends ".NS" if the symbol has no exchange suffix.
func FetchPrice(symbol string) (*PriceResult, error) {
	yahooSymbol := symbol
	if !strings.Contains(symbol, ".") {
		yahooSymbol = symbol + ".NS"
	}

	crumb, cookies, err := yahoo.GetCrumb()
	if err != nil {
		return nil, fmt.Errorf("yahoo price: %w", err)
	}

	reqURL := fmt.Sprintf("%s/%s?modules=price&crumb=%s", yahooSummaryURL, url.PathEscape(yahooSymbol), url.QueryEscape(crumb))

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo price fetch: %w", err)
	}
	defer resp.Body.Close()

	var result yahooPriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("yahoo price decode: %w", err)
	}

	if len(result.QuoteSummary.Result) == 0 {
		return nil, fmt.Errorf("no price data for symbol %q", symbol)
	}

	p := result.QuoteSummary.Result[0].Price
	return &PriceResult{
		Symbol:    yahooSymbol,
		Price:     p.RegularMarketPrice.Raw,
		Currency:  p.Currency,
		ShortName: p.ShortName,
	}, nil
}
