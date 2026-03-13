package backtest

// BuyAndHold buys on the first trading day and sells on the last.
type BuyAndHold struct{}

func (BuyAndHold) Name() string { return "buy_and_hold" }

func (BuyAndHold) GenerateSignals(prices []OHLCV) []Signal {
	if len(prices) == 0 {
		return nil
	}
	return []Signal{
		{Date: prices[0].Date, Action: Buy, Price: prices[0].Close},
		{Date: prices[len(prices)-1].Date, Action: Sell, Price: prices[len(prices)-1].Close},
	}
}
