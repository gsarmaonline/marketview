package indicators

// Signal represents the investment signal from an indicator.
type Signal int

const (
	Bearish Signal = -1
	Neutral Signal = 0
	Bullish Signal = 1
)

func (s Signal) String() string {
	switch s {
	case Bullish:
		return "bullish"
	case Bearish:
		return "bearish"
	default:
		return "neutral"
	}
}

// IndicatorResult holds the fetched data and its interpretation.
type IndicatorResult struct {
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Signal      string  `json:"signal"`
	Description string  `json:"description"`
}

// Indicator fetches a market signal and returns a scored result.
type Indicator interface {
	Name() string
	Fetch() (IndicatorResult, error)
}
