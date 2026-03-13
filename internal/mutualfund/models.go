package mutualfund

// SearchResult is a single fund returned from a search query.
type SearchResult struct {
	SchemeCode int    `json:"schemeCode"`
	SchemeName string `json:"schemeName"`
}

// FundDetails is the full response for a single mutual fund.
type FundDetails struct {
	SchemeCode     int        `json:"schemeCode"`
	SchemeName     string     `json:"schemeName"`
	FundHouse      string     `json:"fundHouse"`
	SchemeType     string     `json:"schemeType"`
	SchemeCategory string     `json:"schemeCategory"`
	LatestNAV      float64    `json:"latestNAV"`
	NAVDate        string     `json:"navDate"`
	NAVHistory     []NAVPoint `json:"navHistory"`
	Holdings       []Holding  `json:"holdings"`
	Stats          *FundStats `json:"stats,omitempty"`
}

// NAVPoint is a single NAV observation.
type NAVPoint struct {
	Date string  `json:"date"`
	NAV  float64 `json:"nav"`
}

// Holding is a single portfolio holding with its allocation percentage.
type Holding struct {
	Name       string  `json:"name"`
	Symbol     string  `json:"symbol,omitempty"`
	Percentage float64 `json:"percentage"` // 0–100
}

// FundInput is a fund reference from the portfolio (name + optional scheme code).
type FundInput struct {
	Name       string `json:"name"`
	SchemeCode int    `json:"schemeCode,omitempty"`
}

// PortfolioFundAnalysis is the per-fund result in a portfolio analysis.
type PortfolioFundAnalysis struct {
	Name       string    `json:"name"`
	SchemeCode int       `json:"schemeCode,omitempty"`
	FundHouse  string    `json:"fundHouse,omitempty"`
	Category   string    `json:"category,omitempty"`
	Holdings   []Holding `json:"holdings"`
}

// FundAllocation shows one fund's allocation in a shared stock.
type FundAllocation struct {
	FundName   string  `json:"fundName"`
	Percentage float64 `json:"percentage"`
}

// StockOverlap describes a stock that appears in two or more portfolio funds.
type StockOverlap struct {
	StockName string           `json:"stockName"`
	Symbol    string           `json:"symbol,omitempty"`
	Funds     []FundAllocation `json:"funds"`
}

// PortfolioAnalysis is the complete result returned by the analyse endpoint.
type PortfolioAnalysis struct {
	Funds           []PortfolioFundAnalysis `json:"funds"`
	Overlaps        []StockOverlap          `json:"overlaps"`
	Recommendations []string                `json:"recommendations"`
}

// FundStats contains additional fund-level statistics.
type FundStats struct {
	AUM               float64 `json:"aum,omitempty"`               // total assets in fund currency
	Yield             float64 `json:"yield,omitempty"`              // trailing yield %
	YTDReturn         float64 `json:"ytdReturn,omitempty"`          // year-to-date return %
	Beta3Year         float64 `json:"beta3Year,omitempty"`          // 3-year beta vs benchmark
	MorningStarRating int     `json:"morningStarRating,omitempty"` // 1–5 stars
	EquityPE          float64 `json:"equityPE,omitempty"`          // weighted avg P/E of holdings
	EquityPB          float64 `json:"equityPB,omitempty"`          // weighted avg P/B of holdings
	StockAllocation   float64 `json:"stockAllocation,omitempty"`   // % in equities
	BondAllocation    float64 `json:"bondAllocation,omitempty"`    // % in bonds
	CashAllocation    float64 `json:"cashAllocation,omitempty"`    // % in cash
	Category          string  `json:"category,omitempty"`          // Morningstar category
	Description       string  `json:"description,omitempty"`       // fund description
}
