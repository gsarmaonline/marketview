package news

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"Mon, 02 Jan 2006 15:04:05 MST", false},  // RFC1123
		{"Mon, 02 Jan 2006 15:04:05 +0000", false}, // RFC1123Z
		{"Mon, 13 Mar 2026 10:30:00 +0530", false},
		{"2026-03-13T10:30:00Z", false},
		{"2026-03-13 10:30:00 +0530", false},
		{"", true},
		{"not a date", true},
		{"13/03/2026", true},
	}
	for _, tt := range tests {
		_, err := parseDate(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseDate(%q) err=%v, wantErr=%v", tt.input, err, tt.wantErr)
		}
	}
}

func TestParseDate_PreservesTime(t *testing.T) {
	// RFC1123Z: "Mon, 02 Jan 2006 15:04:05 -0700"
	input := "Fri, 13 Mar 2026 10:30:00 +0000"
	got, err := parseDate(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Year() != 2026 || got.Month() != time.March || got.Day() != 13 {
		t.Errorf("unexpected date: %v", got)
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"no tags here", "no tags here"},
		{"<b>bold</b>", "bold"},
		{"<p>Hello <em>world</em></p>", "Hello world"},
		{"<a href='http://x.com'>link</a> text", "link text"},
		{"  spaces  ", "spaces"},
		{"a  b   c", "a b c"}, // collapses multiple spaces
		{"<br/>", ""},
		{"", ""},
		{"<!-- comment -->", ""},
	}
	for _, tt := range tests {
		got := stripHTML(tt.input)
		if got != tt.want {
			t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFetch_MockServer(t *testing.T) {
	rssBody := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>
<item>
  <title>Market Update</title>
  <link>https://example.com/article1</link>
  <description>Markets rose today</description>
  <pubDate>Fri, 13 Mar 2026 10:00:00 +0000</pubDate>
</item>
<item>
  <title>Another Story</title>
  <link>https://example.com/article2</link>
  <description>More market news</description>
  <pubDate>Fri, 13 Mar 2026 09:00:00 +0000</pubDate>
</item>
</channel></rss>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprint(w, rssBody)
	}))
	defer ts.Close()

	origSources := sources
	origClient := httpClient
	defer func() {
		sources = origSources
		httpClient = origClient
	}()

	sources = []source{{name: "TestSource", url: ts.URL}}
	httpClient = ts.Client()

	items, err := Fetch(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Source != "TestSource" {
		t.Errorf("expected source %q, got %q", "TestSource", items[0].Source)
	}
	// Items should be sorted newest first.
	if items[0].Title != "Market Update" {
		t.Errorf("expected newest item first, got %q", items[0].Title)
	}
}

func TestFetch_Limit(t *testing.T) {
	rssBody := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>
<item><title>A</title><link>https://example.com/a</link><description/><pubDate>Fri, 13 Mar 2026 10:00:00 +0000</pubDate></item>
<item><title>B</title><link>https://example.com/b</link><description/><pubDate>Fri, 13 Mar 2026 09:00:00 +0000</pubDate></item>
<item><title>C</title><link>https://example.com/c</link><description/><pubDate>Fri, 13 Mar 2026 08:00:00 +0000</pubDate></item>
</channel></rss>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, rssBody)
	}))
	defer ts.Close()

	origSources := sources
	origClient := httpClient
	defer func() {
		sources = origSources
		httpClient = origClient
	}()

	sources = []source{{name: "TestSource", url: ts.URL}}
	httpClient = ts.Client()

	items, err := Fetch(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items after limit, got %d", len(items))
	}
}

func TestFetch_AllSourcesFail(t *testing.T) {
	origSources := sources
	origClient := httpClient
	defer func() {
		sources = origSources
		httpClient = origClient
	}()

	sources = []source{{name: "BadSource", url: "http://127.0.0.1:0/bad"}}
	httpClient = &http.Client{Timeout: 1 * time.Millisecond}

	_, err := Fetch(10)
	if err == nil {
		t.Error("expected error when all sources fail, got nil")
	}
}

func TestFetch_PartialFailure(t *testing.T) {
	goodRSS := `<?xml version="1.0"?><rss version="2.0"><channel>
<item><title>Good</title><link>https://example.com/g</link><description/><pubDate>Fri, 13 Mar 2026 10:00:00 +0000</pubDate></item>
</channel></rss>`

	goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goodRSS)
	}))
	defer goodServer.Close()

	origSources := sources
	origClient := httpClient
	defer func() {
		sources = origSources
		httpClient = origClient
	}()

	sources = []source{
		{name: "BadSource", url: "http://127.0.0.1:0/bad"},
		{name: "GoodSource", url: goodServer.URL},
	}
	httpClient = goodServer.Client()

	items, err := Fetch(10)
	if err != nil {
		t.Fatalf("unexpected error with partial failure: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item from good source, got %d", len(items))
	}
}
