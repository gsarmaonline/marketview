package news

import (
	"sync"
	"testing"
)

func TestStore_GetUnknownSymbol(t *testing.T) {
	s := NewStore()
	items := s.Get("UNKNOWN")
	if items == nil {
		t.Error("Get should return empty slice, not nil")
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items for unknown symbol, got %d", len(items))
	}
}

func TestStore_Ingest_Basic(t *testing.T) {
	s := NewStore()
	s.Ingest("RELIANCE", []NewsItem{
		{Title: "Story 1", Link: "https://example.com/1"},
		{Title: "Story 2", Link: "https://example.com/2"},
	})

	items := s.Get("RELIANCE")
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestStore_Ingest_NormalisesSymbol(t *testing.T) {
	s := NewStore()
	s.Ingest("reliance", []NewsItem{{Title: "Story", Link: "https://example.com/r"}})

	// Both lowercase and uppercase lookups should work.
	if len(s.Get("RELIANCE")) != 1 {
		t.Error("expected item when looking up with uppercase after lowercase ingest")
	}
	if len(s.Get("reliance")) != 1 {
		t.Error("expected item when looking up with lowercase after lowercase ingest")
	}
}

func TestStore_Ingest_TagsSymbol(t *testing.T) {
	s := NewStore()
	s.Ingest("TCS", []NewsItem{{Title: "TCS news", Link: "https://example.com/tcs"}})

	items := s.Get("TCS")
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Symbol != "TCS" {
		t.Errorf("expected symbol %q, got %q", "TCS", items[0].Symbol)
	}
}

func TestStore_Ingest_Deduplication(t *testing.T) {
	s := NewStore()
	item := NewsItem{Title: "Duplicate", Link: "https://example.com/dup"}
	s.Ingest("INFY", []NewsItem{item})
	s.Ingest("INFY", []NewsItem{item})

	if len(s.Get("INFY")) != 1 {
		t.Errorf("expected 1 item after duplicate ingest, got %d", len(s.Get("INFY")))
	}
}

func TestStore_Ingest_AppendNewItems(t *testing.T) {
	s := NewStore()
	s.Ingest("WIPRO", []NewsItem{{Title: "First", Link: "https://example.com/1"}})
	s.Ingest("WIPRO", []NewsItem{{Title: "Second", Link: "https://example.com/2"}})

	if len(s.Get("WIPRO")) != 2 {
		t.Errorf("expected 2 items after two distinct ingests, got %d", len(s.Get("WIPRO")))
	}
}

func TestStore_Ingest_EmptySymbolIgnored(t *testing.T) {
	s := NewStore()
	s.Ingest("", []NewsItem{{Title: "Story", Link: "https://example.com/x"}})
	s.Ingest("   ", []NewsItem{{Title: "Story", Link: "https://example.com/x"}})

	if len(s.Symbols()) != 0 {
		t.Errorf("expected no symbols after ingesting with empty symbol, got %v", s.Symbols())
	}
}

func TestStore_Ingest_EmptyItemsIgnored(t *testing.T) {
	s := NewStore()
	s.Ingest("SBIN", nil)
	s.Ingest("SBIN", []NewsItem{})

	if len(s.Get("SBIN")) != 0 {
		t.Errorf("expected 0 items after ingesting empty slices, got %d", len(s.Get("SBIN")))
	}
}

func TestStore_Replace_Basic(t *testing.T) {
	s := NewStore()
	s.Ingest("HDFC", []NewsItem{
		{Title: "Old", Link: "https://example.com/old"},
	})
	s.Replace("HDFC", []NewsItem{
		{Title: "New 1", Link: "https://example.com/new1"},
		{Title: "New 2", Link: "https://example.com/new2"},
	})

	items := s.Get("HDFC")
	if len(items) != 2 {
		t.Fatalf("expected 2 items after Replace, got %d", len(items))
	}
	if items[0].Title != "New 1" || items[1].Title != "New 2" {
		t.Error("Replace did not overwrite existing items")
	}
}

func TestStore_Replace_EmptySymbolIgnored(t *testing.T) {
	s := NewStore()
	s.Replace("", []NewsItem{{Title: "Story", Link: "https://example.com/x"}})
	if len(s.Symbols()) != 0 {
		t.Error("Replace with empty symbol should be a no-op")
	}
}

func TestStore_Replace_TagsSymbol(t *testing.T) {
	s := NewStore()
	s.Replace("axis", []NewsItem{{Title: "Story", Link: "https://example.com/ax"}})

	items := s.Get("AXIS")
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Symbol != "AXIS" {
		t.Errorf("expected symbol AXIS, got %q", items[0].Symbol)
	}
}

func TestStore_Symbols(t *testing.T) {
	s := NewStore()
	s.Ingest("RELIANCE", []NewsItem{{Link: "https://a.com/1"}})
	s.Ingest("TCS", []NewsItem{{Link: "https://a.com/2"}})
	s.Ingest("INFY", []NewsItem{{Link: "https://a.com/3"}})

	syms := s.Symbols()
	if len(syms) != 3 {
		t.Fatalf("expected 3 symbols, got %d: %v", len(syms), syms)
	}

	seen := make(map[string]bool)
	for _, sym := range syms {
		seen[sym] = true
	}
	for _, want := range []string{"RELIANCE", "TCS", "INFY"} {
		if !seen[want] {
			t.Errorf("expected symbol %q in Symbols(), not found", want)
		}
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	s := NewStore()
	var wg sync.WaitGroup
	symbols := []string{"A", "B", "C", "D", "E"}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sym := symbols[i%len(symbols)]
			s.Ingest(sym, []NewsItem{{Link: "https://example.com/" + sym}})
			_ = s.Get(sym)
			_ = s.Symbols()
		}(i)
	}
	wg.Wait()
}
