package news

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type NewsItem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Link        string `json:"link"`
	PublishedAt string `json:"publishedAt"`
	Source      string `json:"source"`
	Symbol      string `json:"symbol,omitempty"` // set for stock-specific news, empty for general news
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssFeed struct {
	Channel rssChannel `xml:"channel"`
}

type source struct {
	name string
	url  string
}

var sources = []source{
	{name: "The Hindu BusinessLine", url: "https://www.thehindubusinessline.com/markets/feeder/default.rss"},
	{name: "NDTV Profit", url: "https://feeds.feedburner.com/ndtvprofit-latest"},
	{name: "Investing.com India", url: "https://in.investing.com/rss/news.rss"},
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func Fetch(limit int) ([]NewsItem, error) {
	var all []NewsItem

	for _, src := range sources {
		items, err := fetchSource(src)
		if err != nil {
			// partial failure is acceptable; log is handled by caller
			continue
		}
		all = append(all, items...)
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("all news sources failed")
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].PublishedAt > all[j].PublishedAt // RFC3339 sorts lexicographically
	})

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}

	return all, nil
}

func fetchSource(src source) ([]NewsItem, error) {
	req, err := http.NewRequest("GET", src.url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MarketView/1.0)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", src.name, err)
	}
	defer resp.Body.Close()

	var feed rssFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("parsing %s RSS: %w", src.name, err)
	}

	items := make([]NewsItem, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		pubAt := ""
		if t, err := parseDate(item.PubDate); err == nil {
			pubAt = t.UTC().Format(time.RFC3339)
		}

		items = append(items, NewsItem{
			Title:       strings.TrimSpace(item.Title),
			Description: stripHTML(item.Description),
			Link:        strings.TrimSpace(item.Link),
			PublishedAt: pubAt,
			Source:      src.name,
		})
	}

	return items, nil
}

var dateFormats = []string{
	time.RFC1123,
	time.RFC1123Z,
	"Mon, 02 Jan 2006 15:04:05 -0700",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02 15:04:05",
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, f := range dateFormats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable date: %s", s)
}

func stripHTML(s string) string {
	out := make([]byte, 0, len(s))
	inTag := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				out = append(out, s[i])
			}
		}
	}
	result := strings.TrimSpace(string(out))
	// Collapse multiple spaces
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}
	return result
}
