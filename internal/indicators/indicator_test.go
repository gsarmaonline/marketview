package indicators

import "testing"

func TestSignal_String(t *testing.T) {
	tests := []struct {
		signal Signal
		want   string
	}{
		{Bullish, "bullish"},
		{Bearish, "bearish"},
		{Neutral, "neutral"},
		{Signal(42), "neutral"},  // unknown values default to neutral
		{Signal(-99), "neutral"}, // large negative also neutral
	}
	for _, tt := range tests {
		got := tt.signal.String()
		if got != tt.want {
			t.Errorf("Signal(%d).String() = %q, want %q", tt.signal, got, tt.want)
		}
	}
}
