package api

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"marketview/internal/news"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// randomSymbol returns a random stock symbol from a small fixed set.
func randomSymbol() string {
	symbols := []string{"HDFCBANK", "RELIANCE", "TCS", "INFY", "WIPRO", "ICICIBANK", "SBIN", "AXISBANK"}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return symbols[r.Intn(len(symbols))]
}

func newTestServer(store *news.Store) *Server {
	s, _ := New(context.Background(), nil, nil, nil, store, nil)
	return s
}

func TestStockNews_EmptyStore(t *testing.T) {
	store := news.NewStore()
	srv := newTestServer(store)

	symbol := randomSymbol()
	req := httptest.NewRequest(http.MethodGet, "/api/news/stock/"+symbol, nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var items []news.NewsItem
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty slice for uninitialised symbol, got %d items", len(items))
	}
}

func TestStockNews_IngestedItems(t *testing.T) {
	store := news.NewStore()
	symbol := randomSymbol()

	ingested := []news.NewsItem{
		{
			Title:       symbol + " Q4 results beat estimates",
			Description: "Net profit rose 18% YoY.",
			Link:        "https://example.com/" + symbol + "/q4",
			PublishedAt: time.Now().UTC().Format(time.RFC3339),
			Source:      "Economic Times",
		},
		{
			Title:       symbol + " acquires subsidiary",
			Description: "Deal valued at Rs 2,000 crore.",
			Link:        "https://example.com/" + symbol + "/acquisition",
			PublishedAt: time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
			Source:      "Moneycontrol",
		},
	}
	store.Ingest(symbol, ingested)

	srv := newTestServer(store)
	req := httptest.NewRequest(http.MethodGet, "/api/news/stock/"+symbol, nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var items []news.NewsItem
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(items) != len(ingested) {
		t.Fatalf("expected %d items, got %d", len(ingested), len(items))
	}

	for _, item := range items {
		if item.Symbol != symbol {
			t.Errorf("expected symbol %q, got %q", symbol, item.Symbol)
		}
	}
}

func TestStockNews_CaseInsensitive(t *testing.T) {
	store := news.NewStore()
	symbol := randomSymbol()

	store.Ingest(symbol, []news.NewsItem{
		{
			Title:  symbol + " news item",
			Link:   "https://example.com/" + symbol,
			Source: "Business Standard",
		},
	})

	srv := newTestServer(store)

	// Query with lowercase symbol — store normalises to uppercase internally.
	lower := ""
	for _, c := range symbol {
		lower += string(c + 32) // convert uppercase ASCII to lowercase
	}

	req := httptest.NewRequest(http.MethodGet, "/api/news/stock/"+lower, nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var items []news.NewsItem
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(items) == 0 {
		t.Error("expected items when querying with lowercase symbol, got none")
	}
}

func TestStockNews_IsolatedPerSymbol(t *testing.T) {
	store := news.NewStore()

	symbolA := randomSymbol()
	// Pick a different symbol for B.
	symbolB := randomSymbol()
	for symbolB == symbolA {
		symbolB = randomSymbol()
	}

	store.Ingest(symbolA, []news.NewsItem{
		{Title: symbolA + " news", Link: "https://example.com/a", Source: "ET"},
	})

	srv := newTestServer(store)
	req := httptest.NewRequest(http.MethodGet, "/api/news/stock/"+symbolB, nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	var items []news.NewsItem
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected no items for %s after ingesting only %s, got %d", symbolB, symbolA, len(items))
	}
}

func TestStockNews_Deduplication(t *testing.T) {
	store := news.NewStore()
	symbol := randomSymbol()

	item := news.NewsItem{
		Title:  symbol + " duplicate item",
		Link:   "https://example.com/" + symbol + "/dup",
		Source: "ET",
	}

	// Ingest the same item twice.
	store.Ingest(symbol, []news.NewsItem{item})
	store.Ingest(symbol, []news.NewsItem{item})

	srv := newTestServer(store)
	req := httptest.NewRequest(http.MethodGet, "/api/news/stock/"+symbol, nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	var items []news.NewsItem
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item after deduplication, got %d", len(items))
	}
}
