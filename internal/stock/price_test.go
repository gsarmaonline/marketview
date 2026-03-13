package stock

import (
	"testing"
)

func TestFetchPrice(t *testing.T) {
	symbols := []string{"RELIANCE", "TCS", "INFY"}
	for _, sym := range symbols {
		t.Run(sym, func(t *testing.T) {
			result, err := FetchPrice(sym)
			if err != nil {
				t.Fatalf("FetchPrice(%q) error: %v", sym, err)
			}
			if result.Price <= 0 {
				t.Errorf("expected positive price, got %f", result.Price)
			}
			t.Logf("%s: ₹%.2f", result.Symbol, result.Price)
		})
	}
}
