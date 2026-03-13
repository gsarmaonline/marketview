package indicators

import "testing"

func TestScorePE(t *testing.T) {
	tests := []struct {
		pe   float64
		want Signal
	}{
		{0, Bullish},
		{15, Bullish},
		{19.99, Bullish},
		{20, Neutral},   // boundary: 20 is not < 20
		{22.5, Neutral},
		{25, Neutral},   // boundary: 25 is not > 25
		{25.01, Bearish},
		{30, Bearish},
		{100, Bearish},
	}
	for _, tt := range tests {
		got := scorePE(tt.pe)
		if got != tt.want {
			t.Errorf("scorePE(%.2f) = %v, want %v", tt.pe, got, tt.want)
		}
	}
}

func TestExtractNiftyPE_Found(t *testing.T) {
	data := niftyPEResponse{
		Data: []struct {
			Symbol string  `json:"symbol"`
			PE     float64 `json:"pe"`
			PB     float64 `json:"pb"`
		}{
			{Symbol: "HDFC BANK", PE: 18.5},
			{Symbol: "NIFTY 50", PE: 22.3},
			{Symbol: "RELIANCE", PE: 25.1},
		},
	}
	pe, err := extractNiftyPE(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pe != 22.3 {
		t.Errorf("extractNiftyPE() = %v, want 22.3", pe)
	}
}

func TestExtractNiftyPE_FirstEntry(t *testing.T) {
	data := niftyPEResponse{
		Data: []struct {
			Symbol string  `json:"symbol"`
			PE     float64 `json:"pe"`
			PB     float64 `json:"pb"`
		}{
			{Symbol: "NIFTY 50", PE: 18.0},
		},
	}
	pe, err := extractNiftyPE(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pe != 18.0 {
		t.Errorf("extractNiftyPE() = %v, want 18.0", pe)
	}
}

func TestExtractNiftyPE_NotFound(t *testing.T) {
	data := niftyPEResponse{
		Data: []struct {
			Symbol string  `json:"symbol"`
			PE     float64 `json:"pe"`
			PB     float64 `json:"pb"`
		}{
			{Symbol: "HDFC BANK", PE: 18.5},
			{Symbol: "RELIANCE", PE: 25.1},
		},
	}
	_, err := extractNiftyPE(data)
	if err == nil {
		t.Error("expected error when NIFTY 50 not found, got nil")
	}
}

func TestExtractNiftyPE_EmptyData(t *testing.T) {
	data := niftyPEResponse{}
	_, err := extractNiftyPE(data)
	if err == nil {
		t.Error("expected error for empty data, got nil")
	}
}

func TestPEDescription(t *testing.T) {
	tests := []struct {
		pe     float64
		signal Signal
		want   string
	}{
		{15, Bullish, "PE of 15.0x is below 20 — market is historically cheap"},
		{19.9, Bullish, "PE of 19.9x is below 20 — market is historically cheap"},
		{30, Bearish, "PE of 30.0x is above 25 — market is historically expensive"},
		{25.1, Bearish, "PE of 25.1x is above 25 — market is historically expensive"},
		{22, Neutral, "PE of 22.0x is in the fair-value range (20–25)"},
		{20, Neutral, "PE of 20.0x is in the fair-value range (20–25)"},
	}
	for _, tt := range tests {
		got := peDescription(tt.pe, tt.signal)
		if got != tt.want {
			t.Errorf("peDescription(%.1f, %v) = %q, want %q", tt.pe, tt.signal, got, tt.want)
		}
	}
}

func TestNiftyPEName(t *testing.T) {
	// nse.Client requires a real network call, so we only test Name() here.
	// Fetch() requires integration testing with a live NSE session.
	ind := &NiftyPE{}
	if got := ind.Name(); got != "NIFTY 50 PE Ratio" {
		t.Errorf("Name() = %q, want %q", got, "NIFTY 50 PE Ratio")
	}
}
