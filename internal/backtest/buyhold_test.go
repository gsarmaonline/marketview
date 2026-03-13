package backtest

import (
	"testing"
	"time"
)

func TestBuyAndHold_Name(t *testing.T) {
	s := BuyAndHold{}
	if s.Name() != "buy_and_hold" {
		t.Error("expected name buy_and_hold")
	}
}

func TestBuyAndHold_EmptyPrices(t *testing.T) {
	signals := (BuyAndHold{}).GenerateSignals(nil)
	if len(signals) != 0 {
		t.Errorf("expected 0 signals for empty prices, got %d", len(signals))
	}
}

func TestBuyAndHold_SinglePrice(t *testing.T) {
	prices := []OHLCV{
		{Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100},
	}
	signals := (BuyAndHold{}).GenerateSignals(prices)
	if len(signals) != 2 {
		t.Fatalf("expected 2 signals for single price, got %d", len(signals))
	}
	if signals[0].Action != Buy {
		t.Errorf("first signal = %s, want BUY", signals[0].Action)
	}
	if signals[1].Action != Sell {
		t.Errorf("second signal = %s, want SELL", signals[1].Action)
	}
}

func TestBuyAndHold_BuysFirstSellsLast(t *testing.T) {
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	mid := time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	prices := []OHLCV{
		{Date: start, Close: 100},
		{Date: mid, Close: 200},
		{Date: end, Close: 150},
	}

	signals := (BuyAndHold{}).GenerateSignals(prices)

	if len(signals) != 2 {
		t.Fatalf("expected 2 signals, got %d", len(signals))
	}
	if signals[0].Action != Buy || !signals[0].Date.Equal(start) {
		t.Errorf("expected BUY on %v, got %s on %v", start, signals[0].Action, signals[0].Date)
	}
	if signals[1].Action != Sell || !signals[1].Date.Equal(end) {
		t.Errorf("expected SELL on %v, got %s on %v", end, signals[1].Action, signals[1].Date)
	}
	if signals[0].Price != 100 {
		t.Errorf("buy price = %f, want 100", signals[0].Price)
	}
	if signals[1].Price != 150 {
		t.Errorf("sell price = %f, want 150", signals[1].Price)
	}
}
