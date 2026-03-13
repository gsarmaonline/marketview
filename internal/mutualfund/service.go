package mutualfund

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
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

// AnalysePortfolio fetches holdings for each input fund and returns overlap and recommendations.
func (s *Service) AnalysePortfolio(inputs []FundInput) (*PortfolioAnalysis, error) {
	result := &PortfolioAnalysis{
		Funds:           []PortfolioFundAnalysis{},
		Overlaps:        []StockOverlap{},
		Recommendations: []string{},
	}

	for _, input := range inputs {
		schemeCode := input.SchemeCode
		if schemeCode == 0 {
			results, err := searchFunds(input.Name)
			if err != nil || len(results) == 0 {
				result.Funds = append(result.Funds, PortfolioFundAnalysis{Name: input.Name, Holdings: []Holding{}})
				continue
			}
			schemeCode = results[0].SchemeCode
		}

		details, err := s.GetDetails(schemeCode)
		if err != nil {
			result.Funds = append(result.Funds, PortfolioFundAnalysis{Name: input.Name, Holdings: []Holding{}})
			continue
		}

		result.Funds = append(result.Funds, PortfolioFundAnalysis{
			Name:       input.Name,
			SchemeCode: schemeCode,
			FundHouse:  details.FundHouse,
			Category:   details.SchemeCategory,
			Holdings:   details.Holdings,
		})
	}

	// Build stock -> funds map to find overlaps.
	type stockEntry struct {
		fundName   string
		percentage float64
		symbol     string
	}
	stockMap := make(map[string][]stockEntry)
	for _, fund := range result.Funds {
		for _, h := range fund.Holdings {
			stockMap[h.Name] = append(stockMap[h.Name], stockEntry{
				fundName:   fund.Name,
				percentage: h.Percentage,
				symbol:     h.Symbol,
			})
		}
	}

	for stockName, entries := range stockMap {
		if len(entries) < 2 {
			continue
		}
		overlap := StockOverlap{
			StockName: stockName,
			Symbol:    entries[0].symbol,
			Funds:     make([]FundAllocation, 0, len(entries)),
		}
		for _, e := range entries {
			overlap.Funds = append(overlap.Funds, FundAllocation{FundName: e.fundName, Percentage: e.percentage})
		}
		result.Overlaps = append(result.Overlaps, overlap)
	}

	// Sort: most-shared stocks first, then by highest combined allocation.
	sort.Slice(result.Overlaps, func(i, j int) bool {
		if len(result.Overlaps[i].Funds) != len(result.Overlaps[j].Funds) {
			return len(result.Overlaps[i].Funds) > len(result.Overlaps[j].Funds)
		}
		var si, sj float64
		for _, f := range result.Overlaps[i].Funds {
			si += f.Percentage
		}
		for _, f := range result.Overlaps[j].Funds {
			sj += f.Percentage
		}
		return si > sj
	})

	result.Recommendations = s.generateRecommendations(result)
	return result, nil
}

func (s *Service) generateRecommendations(analysis *PortfolioAnalysis) []string {
	var recs []string

	// Pairwise overlap score between every pair of funds.
	for i := 0; i < len(analysis.Funds); i++ {
		for j := i + 1; j < len(analysis.Funds); j++ {
			fi, fj := analysis.Funds[i], analysis.Funds[j]
			holdingsI := make(map[string]float64)
			for _, h := range fi.Holdings {
				holdingsI[h.Name] = h.Percentage
			}
			var overlap float64
			for _, h := range fj.Holdings {
				if pct, ok := holdingsI[h.Name]; ok {
					if h.Percentage < pct {
						overlap += h.Percentage
					} else {
						overlap += pct
					}
				}
			}
			if overlap >= 40 {
				recs = append(recs, fmt.Sprintf(
					"%.0f%% stock overlap between \"%s\" and \"%s\" — consider consolidating into one fund.",
					overlap, fi.Name, fj.Name,
				))
			} else if overlap >= 20 {
				recs = append(recs, fmt.Sprintf(
					"%.0f%% common holdings between \"%s\" and \"%s\" — review for redundancy.",
					overlap, fi.Name, fj.Name,
				))
			}
		}
	}

	// Same-category concentration.
	catCount := make(map[string]int)
	for _, f := range analysis.Funds {
		if f.Category != "" {
			catCount[f.Category]++
		}
	}
	for cat, count := range catCount {
		if count >= 2 {
			recs = append(recs, fmt.Sprintf(
				"%d funds in the \"%s\" category — consolidating may simplify your portfolio.",
				count, cat,
			))
		}
	}

	// Missing diversification.
	if len(analysis.Funds) > 0 {
		hasMidSmall, hasDebt, hasIntl := false, false, false
		for _, f := range analysis.Funds {
			cat := strings.ToLower(f.Category)
			if strings.Contains(cat, "mid") || strings.Contains(cat, "small") {
				hasMidSmall = true
			}
			if strings.Contains(cat, "debt") || strings.Contains(cat, "bond") ||
				strings.Contains(cat, "liquid") || strings.Contains(cat, "income") {
				hasDebt = true
			}
			if strings.Contains(cat, "international") || strings.Contains(cat, "global") ||
				strings.Contains(cat, "overseas") || strings.Contains(cat, "world") {
				hasIntl = true
			}
		}
		if !hasMidSmall {
			recs = append(recs, "No Mid/Small Cap fund detected — consider adding one for higher long-term growth potential.")
		}
		if !hasDebt {
			recs = append(recs, "No debt or bond fund detected — consider adding one for stability and rebalancing flexibility.")
		}
		if len(analysis.Funds) >= 3 && !hasIntl {
			recs = append(recs, "Consider an international or global fund for geographic diversification beyond India.")
		}
	}

	if len(recs) == 0 && len(analysis.Funds) > 0 {
		recs = append(recs, "Portfolio looks well-diversified with minimal overlap across mutual funds.")
	}

	return recs
}
