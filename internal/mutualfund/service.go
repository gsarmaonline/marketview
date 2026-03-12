package mutualfund

import (
	"fmt"
	"log"
	"strconv"
)

const navHistoryLimit = 30

// Service combines mfapi.in and Yahoo Finance to provide mutual fund data.
type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Search(query string) ([]SearchResult, error) {
	return searchFunds(query)
}

func (s *Service) GetDetails(schemeCode int) (*FundDetails, error) {
	meta, err := fetchFundMeta(schemeCode)
	if err != nil {
		return nil, fmt.Errorf("fetching fund meta: %w", err)
	}

	details := &FundDetails{
		SchemeCode:     meta.Meta.SchemeCode,
		SchemeName:     meta.Meta.SchemeName,
		FundHouse:      meta.Meta.FundHouse,
		SchemeType:     meta.Meta.SchemeType,
		SchemeCategory: meta.Meta.SchemeCategory,
		Holdings:       []Holding{},
	}

	limit := navHistoryLimit
	if len(meta.Data) < limit {
		limit = len(meta.Data)
	}
	details.NAVHistory = make([]NAVPoint, 0, limit)
	for i := 0; i < limit; i++ {
		d := meta.Data[i]
		nav, err := strconv.ParseFloat(d.NAV, 64)
		if err != nil {
			continue
		}
		details.NAVHistory = append(details.NAVHistory, NAVPoint{Date: d.Date, NAV: nav})
	}
	if len(details.NAVHistory) > 0 {
		details.LatestNAV = details.NAVHistory[0].NAV
		details.NAVDate = details.NAVHistory[0].Date
	}

	// Enrich with Yahoo Finance data (holdings + stats). Non-fatal if unavailable.
	symbol, err := findYahooSymbol(meta.Meta.SchemeName)
	if err != nil {
		log.Printf("yahoo symbol lookup failed for %q: %v", meta.Meta.SchemeName, err)
		return details, nil
	}
	if symbol == "" {
		return details, nil
	}

	yahooData, err := fetchYahooHoldings(symbol)
	if err != nil {
		log.Printf("yahoo holdings fetch failed for %q: %v", symbol, err)
		return details, nil
	}
	if yahooData != nil {
		details.Holdings = yahooData.Holdings
		details.Stats = &yahooData.Stats
	}

	return details, nil
}
