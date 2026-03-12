package mutualfund

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const mfAPIBase = "https://api.mfapi.in/mf"

type mfapiSearchResult struct {
	SchemeCode int    `json:"schemeCode"`
	SchemeName string `json:"schemeName"`
}

type mfapiSchemeDetail struct {
	Meta struct {
		FundHouse      string `json:"fund_house"`
		SchemeType     string `json:"scheme_type"`
		SchemeCategory string `json:"scheme_category"`
		SchemeCode     int    `json:"scheme_code"`
		SchemeName     string `json:"scheme_name"`
	} `json:"meta"`
	Data []struct {
		Date string `json:"date"`
		NAV  string `json:"nav"`
	} `json:"data"`
}

func searchFunds(query string) ([]SearchResult, error) {
	resp, err := http.Get(mfAPIBase + "/search?q=" + url.QueryEscape(query))
	if err != nil {
		return nil, fmt.Errorf("mfapi search: %w", err)
	}
	defer resp.Body.Close()

	var raw []mfapiSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("mfapi search decode: %w", err)
	}

	results := make([]SearchResult, len(raw))
	for i, r := range raw {
		results[i] = SearchResult{SchemeCode: r.SchemeCode, SchemeName: r.SchemeName}
	}
	return results, nil
}

func fetchFundMeta(schemeCode int) (*mfapiSchemeDetail, error) {
	resp, err := http.Get(fmt.Sprintf("%s/%d", mfAPIBase, schemeCode))
	if err != nil {
		return nil, fmt.Errorf("mfapi fetch: %w", err)
	}
	defer resp.Body.Close()

	var detail mfapiSchemeDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, fmt.Errorf("mfapi decode: %w", err)
	}
	return &detail, nil
}
