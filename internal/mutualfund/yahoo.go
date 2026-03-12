package mutualfund

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const (
	yahooSearchURL  = "https://query1.finance.yahoo.com/v1/finance/search"
	yahooSummaryURL = "https://query1.finance.yahoo.com/v10/finance/quoteSummary"
)

// yahooValue is a Yahoo Finance wrapped numeric value.
type yahooValue struct {
	Raw float64 `json:"raw"`
}

type yahooSearchResponse struct {
	Quotes []struct {
		Symbol    string `json:"symbol"`
		Shortname string `json:"shortname"`
		QuoteType string `json:"quoteType"`
	} `json:"quotes"`
}

type yahooSummaryResponse struct {
	QuoteSummary struct {
		Result []struct {
			TopHoldings struct {
				Holdings []struct {
					Name           string     `json:"name"`
					Symbol         string     `json:"symbol"`
					HoldingPercent yahooValue `json:"holdingPercent"`
				} `json:"holdings"`
				CashPosition  yahooValue `json:"cashPosition"`
				StockPosition yahooValue `json:"stockPosition"`
				BondPosition  yahooValue `json:"bondPosition"`
				EquityHoldings struct {
					PriceToEarnings yahooValue `json:"priceToEarnings"`
					PriceToBook     yahooValue `json:"priceToBook"`
				} `json:"equityHoldings"`
			} `json:"topHoldings"`
			FundProfile struct {
				CategoryName string `json:"categoryName"`
				Family       string `json:"family"`
				Description  string `json:"description"`
			} `json:"fundProfile"`
			DefaultKeyStatistics struct {
				TotalAssets              yahooValue `json:"totalAssets"`
				Yield                    yahooValue `json:"yield"`
				YtdReturn                yahooValue `json:"ytdReturn"`
				Beta3Year                yahooValue `json:"beta3Year"`
				MorningStarOverallRating yahooValue `json:"morningStarOverallRating"`
			} `json:"defaultKeyStatistics"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"quoteSummary"`
}

type yahooFundData struct {
	Holdings []Holding
	Stats    FundStats
}

func findYahooSymbol(fundName string) (string, error) {
	reqURL := fmt.Sprintf(
		"%s?q=%s&country=IN&lang=en-IN&newsCount=0&enableFuzzyQuery=false&quoteType=MUTUALFUND",
		yahooSearchURL, url.QueryEscape(fundName),
	)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("yahoo search: %w", err)
	}
	defer resp.Body.Close()

	var result yahooSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("yahoo search decode: %w", err)
	}

	for _, q := range result.Quotes {
		if q.QuoteType == "MUTUALFUND" {
			return q.Symbol, nil
		}
	}
	return "", nil
}

func fetchYahooHoldings(symbol string) (*yahooFundData, error) {
	reqURL := fmt.Sprintf(
		"%s/%s?modules=topHoldings,fundProfile,defaultKeyStatistics",
		yahooSummaryURL, url.PathEscape(symbol),
	)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo quoteSummary: %w", err)
	}
	defer resp.Body.Close()

	var result yahooSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("yahoo decode: %w", err)
	}

	if len(result.QuoteSummary.Result) == 0 {
		return nil, nil
	}

	r := result.QuoteSummary.Result[0]

	holdings := make([]Holding, len(r.TopHoldings.Holdings))
	for i, h := range r.TopHoldings.Holdings {
		holdings[i] = Holding{
			Name:       h.Name,
			Symbol:     h.Symbol,
			Percentage: h.HoldingPercent.Raw * 100,
		}
	}

	stats := r.DefaultKeyStatistics
	return &yahooFundData{
		Holdings: holdings,
		Stats: FundStats{
			AUM:               stats.TotalAssets.Raw,
			Yield:             stats.Yield.Raw * 100,
			YTDReturn:         stats.YtdReturn.Raw * 100,
			Beta3Year:         stats.Beta3Year.Raw,
			MorningStarRating: int(stats.MorningStarOverallRating.Raw),
			EquityPE:          r.TopHoldings.EquityHoldings.PriceToEarnings.Raw,
			EquityPB:          r.TopHoldings.EquityHoldings.PriceToBook.Raw,
			StockAllocation:   r.TopHoldings.StockPosition.Raw * 100,
			BondAllocation:    r.TopHoldings.BondPosition.Raw * 100,
			CashAllocation:    r.TopHoldings.CashPosition.Raw * 100,
			Category:          r.FundProfile.CategoryName,
			Description:       r.FundProfile.Description,
		},
	}, nil
}
