package news

import (
	"context"
	"log"
	"strings"
	"time"
)

// StockKeyword maps a stock symbol to the keywords that identify it in news text.
type StockKeyword struct {
	Symbol   string
	Keywords []string
}

// Ingester periodically fetches news, matches articles to stocks, and stores them.
type Ingester struct {
	store    *Store
	stocks   []StockKeyword
	interval time.Duration
}

// NiftyStocks is a list of Nifty 50 + major NSE-listed companies with common name aliases.
var NiftyStocks = []StockKeyword{
	{Symbol: "RELIANCE", Keywords: []string{"reliance industries", "reliance jio", "mukesh ambani", "reliance retail"}},
	{Symbol: "TCS", Keywords: []string{"tata consultancy", " tcs "}},
	{Symbol: "INFY", Keywords: []string{"infosys"}},
	{Symbol: "HDFCBANK", Keywords: []string{"hdfc bank"}},
	{Symbol: "ICICIBANK", Keywords: []string{"icici bank"}},
	{Symbol: "HINDUNILVR", Keywords: []string{"hindustan unilever", "hul"}},
	{Symbol: "ITC", Keywords: []string{" itc "}},
	{Symbol: "SBIN", Keywords: []string{"state bank of india", " sbi "}},
	{Symbol: "BAJFINANCE", Keywords: []string{"bajaj finance"}},
	{Symbol: "BHARTIARTL", Keywords: []string{"bharti airtel", "airtel"}},
	{Symbol: "KOTAKBANK", Keywords: []string{"kotak mahindra bank", "kotak bank"}},
	{Symbol: "LT", Keywords: []string{"larsen & toubro", "larsen and toubro", "l&t"}},
	{Symbol: "ASIANPAINT", Keywords: []string{"asian paints"}},
	{Symbol: "AXISBANK", Keywords: []string{"axis bank"}},
	{Symbol: "MARUTI", Keywords: []string{"maruti suzuki", "maruti"}},
	{Symbol: "SUNPHARMA", Keywords: []string{"sun pharmaceutical", "sun pharma"}},
	{Symbol: "TITAN", Keywords: []string{"titan company", "tanishq"}},
	{Symbol: "ULTRACEMCO", Keywords: []string{"ultratech cement"}},
	{Symbol: "NESTLEIND", Keywords: []string{"nestle india"}},
	{Symbol: "WIPRO", Keywords: []string{"wipro"}},
	{Symbol: "HCLTECH", Keywords: []string{"hcl technologies", "hcl tech"}},
	{Symbol: "TECHM", Keywords: []string{"tech mahindra"}},
	{Symbol: "ONGC", Keywords: []string{" ongc ", "oil and natural gas"}},
	{Symbol: "POWERGRID", Keywords: []string{"power grid corporation"}},
	{Symbol: "NTPC", Keywords: []string{" ntpc "}},
	{Symbol: "TATAMOTORS", Keywords: []string{"tata motors", "jaguar land rover"}},
	{Symbol: "TATASTEEL", Keywords: []string{"tata steel"}},
	{Symbol: "JSWSTEEL", Keywords: []string{"jsw steel"}},
	{Symbol: "COALINDIA", Keywords: []string{"coal india"}},
	{Symbol: "INDUSINDBK", Keywords: []string{"indusind bank"}},
	{Symbol: "M&M", Keywords: []string{"mahindra & mahindra", "mahindra and mahindra"}},
	{Symbol: "CIPLA", Keywords: []string{"cipla"}},
	{Symbol: "DRREDDY", Keywords: []string{"dr. reddy", "dr reddy"}},
	{Symbol: "GRASIM", Keywords: []string{"grasim industries"}},
	{Symbol: "HINDALCO", Keywords: []string{"hindalco industries"}},
	{Symbol: "ADANIPORTS", Keywords: []string{"adani ports", "adani enterprises", "adani group", "gautam adani"}},
	{Symbol: "ADANIENT", Keywords: []string{"adani enterprises"}},
	{Symbol: "BAJAJFINSV", Keywords: []string{"bajaj finserv"}},
	{Symbol: "BAJAJ-AUTO", Keywords: []string{"bajaj auto"}},
	{Symbol: "HEROMOTOCO", Keywords: []string{"hero motocorp"}},
	{Symbol: "BRITANNIA", Keywords: []string{"britannia industries"}},
	{Symbol: "EICHERMOT", Keywords: []string{"eicher motors", "royal enfield"}},
	{Symbol: "DIVISLAB", Keywords: []string{"divi's laboratories", "divi's lab"}},
	{Symbol: "APOLLOHOSP", Keywords: []string{"apollo hospitals"}},
	{Symbol: "BPCL", Keywords: []string{" bpcl ", "bharat petroleum"}},
	{Symbol: "IOC", Keywords: []string{"indian oil corporation", " iocl "}},
	{Symbol: "SHREECEM", Keywords: []string{"shree cement"}},
	{Symbol: "SBILIFE", Keywords: []string{"sbi life insurance"}},
	{Symbol: "HDFCLIFE", Keywords: []string{"hdfc life insurance"}},
	{Symbol: "ICICIPRULI", Keywords: []string{"icici prudential life"}},
	{Symbol: "BANKBARODA", Keywords: []string{"bank of baroda"}},
	{Symbol: "PNBBANK", Keywords: []string{"punjab national bank", " pnb "}},
	{Symbol: "CANBK", Keywords: []string{"canara bank"}},
	{Symbol: "VEDL", Keywords: []string{"vedanta limited", "vedanta"}},
	{Symbol: "TATACONSUM", Keywords: []string{"tata consumer products"}},
	{Symbol: "PIDILITIND", Keywords: []string{"pidilite industries", "fevicol"}},
	{Symbol: "HAVELLS", Keywords: []string{"havells india"}},
	{Symbol: "GODREJCP", Keywords: []string{"godrej consumer products"}},
	{Symbol: "DABUR", Keywords: []string{"dabur india"}},
	{Symbol: "MARICO", Keywords: []string{"marico limited"}},
	{Symbol: "COLPAL", Keywords: []string{"colgate-palmolive", "colgate palmolive"}},
	{Symbol: "ZOMATO", Keywords: []string{"zomato"}},
	{Symbol: "NYKAA", Keywords: []string{"nykaa", "fss beauty"}},
	{Symbol: "PAYTM", Keywords: []string{"paytm", "one97 communications"}},
	{Symbol: "POLICYBZR", Keywords: []string{"policybazaar", "pb fintech"}},
	{Symbol: "DMART", Keywords: []string{"avenue supermarts", "d-mart", "dmart"}},
	{Symbol: "IRFC", Keywords: []string{"indian railway finance"}},
	{Symbol: "IRCTC", Keywords: []string{"irctc", "indian railway catering"}},
}

// NewIngester creates an ingester that runs on the given interval.
func NewIngester(store *Store, stocks []StockKeyword, interval time.Duration) *Ingester {
	return &Ingester{store: store, stocks: stocks, interval: interval}
}

// Start launches the ingestion loop in a background goroutine.
// It runs immediately on start and then on every tick.
func (ing *Ingester) Start(ctx context.Context) {
	go func() {
		log.Println("news ingester: starting")
		ing.ingest()

		ticker := time.NewTicker(ing.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("news ingester: stopped")
				return
			case <-ticker.C:
				ing.ingest()
			}
		}
	}()
}

func (ing *Ingester) ingest() {
	items, err := Fetch(100)
	if err != nil {
		log.Printf("news ingester: fetch error: %v", err)
		return
	}

	bySymbol := make(map[string][]NewsItem)
	for _, item := range items {
		for _, sym := range ing.matchSymbols(item) {
			bySymbol[sym] = append(bySymbol[sym], item)
		}
	}

	total := 0
	for sym, stockItems := range bySymbol {
		ing.store.Ingest(sym, stockItems)
		total += len(stockItems)
	}

	log.Printf("news ingester: processed %d articles, matched %d stock-article pairs across %d symbols",
		len(items), total, len(bySymbol))
}

// matchSymbols returns the stock symbols that are mentioned in the news item.
func (ing *Ingester) matchSymbols(item NewsItem) []string {
	text := strings.ToLower(item.Title + " " + item.Description)
	seen := make(map[string]bool)
	var matched []string

	for _, stock := range ing.stocks {
		if seen[stock.Symbol] {
			continue
		}
		for _, kw := range stock.Keywords {
			if strings.Contains(text, strings.ToLower(kw)) {
				matched = append(matched, stock.Symbol)
				seen[stock.Symbol] = true
				break
			}
		}
	}

	return matched
}
