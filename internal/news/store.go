package news

import (
	"strings"
	"sync"
)

// Store is an in-memory pipeline for stock-level news.
// External ingestion sources call Ingest to push items; the API layer calls Get to retrieve them.
// This acts as the central hub for the stock news pipeline — any future source
// (RSS scrapers, webhooks, scheduled jobs) feeds into this store.
type Store struct {
	mu    sync.RWMutex
	items map[string][]NewsItem // keyed by uppercase stock symbol
}

func NewStore() *Store {
	return &Store{items: make(map[string][]NewsItem)}
}

// Ingest adds or replaces news items for a given stock symbol.
// The symbol is normalised to uppercase. Duplicate items (same Link) are deduplicated.
func (s *Store) Ingest(symbol string, items []NewsItem) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if sym == "" || len(items) == 0 {
		return
	}

	// Tag each item with the symbol.
	tagged := make([]NewsItem, len(items))
	for i, it := range items {
		it.Symbol = sym
		tagged[i] = it
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing := s.items[sym]
	seen := make(map[string]struct{}, len(existing))
	for _, it := range existing {
		seen[it.Link] = struct{}{}
	}

	for _, it := range tagged {
		if _, dup := seen[it.Link]; !dup {
			existing = append(existing, it)
			seen[it.Link] = struct{}{}
		}
	}

	s.items[sym] = existing
}

// Replace sets the news items for a symbol, discarding any previously stored items.
func (s *Store) Replace(symbol string, items []NewsItem) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if sym == "" {
		return
	}

	tagged := make([]NewsItem, len(items))
	for i, it := range items {
		it.Symbol = sym
		tagged[i] = it
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[sym] = tagged
}

// Get returns news items for the given symbol (normalised to uppercase).
// Returns an empty slice (never nil) if no news is stored.
func (s *Store) Get(symbol string) []NewsItem {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := s.items[sym]
	if items == nil {
		return []NewsItem{}
	}
	return items
}

// Symbols returns all symbols that currently have news stored.
func (s *Store) Symbols() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	syms := make([]string, 0, len(s.items))
	for k := range s.items {
		syms = append(syms, k)
	}
	return syms
}
