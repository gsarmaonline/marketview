package backtest

import "math"

// Trade represents a completed round-trip (buy → sell).
type Trade struct {
	EntryDate  string  `json:"entry_date"`
	EntryPrice float64 `json:"entry_price"`
	ExitDate   string  `json:"exit_date"`
	ExitPrice  float64 `json:"exit_price"`
	Shares     float64 `json:"shares"`
	PnL        float64 `json:"pnl"`
	ReturnPct  float64 `json:"return_pct"`
}

// EquityPoint is one data point in the equity curve.
type EquityPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// Metrics holds performance statistics for a backtest run.
type Metrics struct {
	TotalReturnPct float64 `json:"total_return_pct"`
	CAGR           float64 `json:"cagr_pct"`
	MaxDrawdownPct float64 `json:"max_drawdown_pct"`
	SharpeRatio    float64 `json:"sharpe_ratio"`
	WinRatePct     float64 `json:"win_rate_pct"`
	TotalTrades    int     `json:"total_trades"`
}

// Result is the full output of a backtest run.
type Result struct {
	Strategy    string        `json:"strategy"`
	Symbol      string        `json:"symbol"`
	From        string        `json:"from"`
	To          string        `json:"to"`
	Capital     float64       `json:"capital"`
	FinalValue  float64       `json:"final_value"`
	Trades      []Trade       `json:"trades"`
	EquityCurve []EquityPoint `json:"equity_curve"`
	Metrics     Metrics       `json:"metrics"`
}

// Run simulates a strategy on historical prices and returns a full backtest result.
func Run(symbol string, prices []OHLCV, strategy Strategy, capital float64) Result {
	signals := strategy.GenerateSignals(prices)
	sigMap := make(map[string]SignalAction, len(signals))
	for _, s := range signals {
		sigMap[s.Date.Format("2006-01-02")] = s.Action
	}

	cash := capital
	shares := 0.0
	var trades []Trade
	var equityCurve []EquityPoint

	type openPosition struct {
		entryDate  string
		entryPrice float64
		shares     float64
	}
	var open *openPosition

	for _, p := range prices {
		dateKey := p.Date.Format("2006-01-02")
		action := sigMap[dateKey]

		switch action {
		case Buy:
			if cash > 0 && shares == 0 {
				shares = math.Floor(cash / p.Close)
				if shares > 0 {
					cash -= shares * p.Close
					open = &openPosition{entryDate: dateKey, entryPrice: p.Close, shares: shares}
				}
			}
		case Sell:
			if shares > 0 && open != nil {
				proceeds := shares * p.Close
				cost := open.shares * open.entryPrice
				pnl := proceeds - cost
				trades = append(trades, Trade{
					EntryDate:  open.entryDate,
					EntryPrice: open.entryPrice,
					ExitDate:   dateKey,
					ExitPrice:  p.Close,
					Shares:     shares,
					PnL:        pnl,
					ReturnPct:  (pnl / cost) * 100,
				})
				cash += proceeds
				shares = 0
				open = nil
			}
		}

		portfolioValue := cash + shares*p.Close
		equityCurve = append(equityCurve, EquityPoint{
			Date:  dateKey,
			Value: portfolioValue,
		})
	}

	finalValue := cash
	if len(equityCurve) > 0 {
		finalValue = equityCurve[len(equityCurve)-1].Value
	}

	from, to := "", ""
	if len(prices) > 0 {
		from = prices[0].Date.Format("2006-01-02")
		to = prices[len(prices)-1].Date.Format("2006-01-02")
	}

	return Result{
		Strategy:    strategy.Name(),
		Symbol:      symbol,
		From:        from,
		To:          to,
		Capital:     capital,
		FinalValue:  finalValue,
		Trades:      trades,
		EquityCurve: equityCurve,
		Metrics:     computeMetrics(capital, finalValue, equityCurve, trades, prices),
	}
}

func computeMetrics(capital, finalValue float64, curve []EquityPoint, trades []Trade, prices []OHLCV) Metrics {
	if len(prices) == 0 {
		return Metrics{}
	}

	totalReturnPct := (finalValue - capital) / capital * 100

	// CAGR
	days := prices[len(prices)-1].Date.Sub(prices[0].Date).Hours() / 24
	years := days / 365.25
	cagr := 0.0
	if years > 0 && finalValue > 0 {
		cagr = (math.Pow(finalValue/capital, 1/years) - 1) * 100
	}

	// Max drawdown
	maxDrawdown := 0.0
	peak := capital
	for _, p := range curve {
		if p.Value > peak {
			peak = p.Value
		}
		dd := (peak - p.Value) / peak * 100
		if dd > maxDrawdown {
			maxDrawdown = dd
		}
	}

	// Sharpe ratio (daily returns, 6% annual risk-free rate for India)
	sharpe := computeSharpe(curve)

	// Win rate
	wins := 0
	for _, t := range trades {
		if t.PnL > 0 {
			wins++
		}
	}
	winRate := 0.0
	if len(trades) > 0 {
		winRate = float64(wins) / float64(len(trades)) * 100
	}

	return Metrics{
		TotalReturnPct: math.Round(totalReturnPct*100) / 100,
		CAGR:           math.Round(cagr*100) / 100,
		MaxDrawdownPct: math.Round(maxDrawdown*100) / 100,
		SharpeRatio:    math.Round(sharpe*100) / 100,
		WinRatePct:     math.Round(winRate*100) / 100,
		TotalTrades:    len(trades),
	}
}

func computeSharpe(curve []EquityPoint) float64 {
	if len(curve) < 2 {
		return 0
	}

	riskFreeDaily := 0.06 / 252

	dailyReturns := make([]float64, 0, len(curve)-1)
	for i := 1; i < len(curve); i++ {
		if curve[i-1].Value == 0 {
			continue
		}
		r := (curve[i].Value - curve[i-1].Value) / curve[i-1].Value
		dailyReturns = append(dailyReturns, r)
	}

	if len(dailyReturns) == 0 {
		return 0
	}

	mean := 0.0
	for _, r := range dailyReturns {
		mean += r
	}
	mean /= float64(len(dailyReturns))

	variance := 0.0
	for _, r := range dailyReturns {
		d := r - mean
		variance += d * d
	}
	variance /= float64(len(dailyReturns))
	std := math.Sqrt(variance)

	if std == 0 {
		return 0
	}

	return (mean - riskFreeDaily) / std * math.Sqrt(252)
}

