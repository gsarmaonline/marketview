package deepresearch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBSEProvider_get_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("expected User-Agent header")
		}
		if r.Header.Get("Referer") == "" {
			t.Error("expected Referer header")
		}
		fmt.Fprint(w, `{"Table":[]}`)
	}))
	defer ts.Close()

	p := &BSEProvider{http: ts.Client()}
	body, err := p.get(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"Table":[]}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestBSEProvider_get_NonOKStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts.Close()

	p := &BSEProvider{http: ts.Client()}
	_, err := p.get(ts.URL)
	if err == nil {
		t.Error("expected error for non-200 status, got nil")
	}
}

func TestBSEProvider_get_ConnectionError(t *testing.T) {
	p := &BSEProvider{http: &http.Client{}}
	_, err := p.get("http://127.0.0.1:0/bad")
	if err == nil {
		t.Error("expected error for unreachable host, got nil")
	}
}

// ── JSON parsing via get() + package-visible types ───────────────────────────

func TestBSEProvider_ParseReportsJSON(t *testing.T) {
	reportJSON := `[
		{"REPORT_YEAR":"2024","PDF_NAME":"Annual Report 2024","PDF_LINK":"http://example.com/2024.pdf"},
		{"REPORT_YEAR":"2023","PDF_NAME":"Annual Report 2023","PDF_LINK":"http://example.com/2023.pdf"}
	]`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, reportJSON)
	}))
	defer ts.Close()

	p := &BSEProvider{http: ts.Client()}
	body, err := p.get(ts.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	var items []bseReportItem
	if err := json.Unmarshal(body, &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ReportYear != "2024" || items[0].PDFName != "Annual Report 2024" {
		t.Errorf("unexpected item[0]: %+v", items[0])
	}
	if items[1].ReportYear != "2023" {
		t.Errorf("unexpected item[1]: %+v", items[1])
	}
}

func TestBSEProvider_ParseReportsJSON_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "not json")
	}))
	defer ts.Close()

	p := &BSEProvider{http: ts.Client()}
	body, err := p.get(ts.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	var items []bseReportItem
	if err := json.Unmarshal(body, &items); err == nil {
		t.Error("expected JSON parse error, got nil")
	}
}

func TestBSEProvider_ParseSearchResult_Empty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Table":[]}`)
	}))
	defer ts.Close()

	p := &BSEProvider{http: ts.Client()}
	body, err := p.get(ts.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	var result bseSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Table) != 0 {
		t.Errorf("expected empty Table, got %d entries", len(result.Table))
	}
}

func TestBSEProvider_ParseSearchResult_Found(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Table":[{"SECURITY_CODE":"500325","SECURITY_NAME":"RELIANCE INDUSTRIES"}]}`)
	}))
	defer ts.Close()

	p := &BSEProvider{http: ts.Client()}
	body, err := p.get(ts.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	var result bseSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Table) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Table))
	}
	if result.Table[0].ScripCode != "500325" {
		t.Errorf("expected scrip code 500325, got %q", result.Table[0].ScripCode)
	}
	if result.Table[0].ScripName != "RELIANCE INDUSTRIES" {
		t.Errorf("expected name RELIANCE INDUSTRIES, got %q", result.Table[0].ScripName)
	}
}
