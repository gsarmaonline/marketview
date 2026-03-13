package deepresearch

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestRouter(svc *Service) *gin.Engine {
	h := NewHandler(svc)
	r := gin.New()
	r.GET("/api/stock/:symbol/deep-research", h.HandleDeepResearch)
	return r
}

func TestHandleDeepResearch_Success(t *testing.T) {
	reports := []AnnualReport{
		{SeqNumber: 1, Issuer: "RELIANCE INDUSTRIES", Year: "2024", Subject: "Annual Report 2024", PDFLink: "http://example.com/2024.pdf"},
		{SeqNumber: 2, Issuer: "RELIANCE INDUSTRIES", Year: "2023", Subject: "Annual Report 2023", PDFLink: "http://example.com/2023.pdf"},
	}
	svc := NewService(&mockProvider{name: "NSE", reports: reports})
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/stock/RELIANCE/deep-research", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result DeepResearch
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Symbol != "RELIANCE" {
		t.Errorf("expected symbol RELIANCE, got %q", result.Symbol)
	}
	if result.AnnualReportsSource != "NSE" {
		t.Errorf("expected source NSE, got %q", result.AnnualReportsSource)
	}
	if len(result.AnnualReports) != 2 {
		t.Errorf("expected 2 reports, got %d", len(result.AnnualReports))
	}
}

func TestHandleDeepResearch_AllProvidersFail(t *testing.T) {
	svc := NewService(&mockProvider{name: "NSE", err: errors.New("NSE unavailable")})
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/stock/TCS/deep-research", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body["error"] == "" {
		t.Error("expected non-empty error message")
	}
}

func TestHandleDeepResearch_SymbolNormalisedInResponse(t *testing.T) {
	svc := NewService(&mockProvider{name: "BSE", reports: []AnnualReport{}})
	r := newTestRouter(svc)

	// Send lowercase symbol — service should normalise it.
	req := httptest.NewRequest(http.MethodGet, "/api/stock/infy/deep-research", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result DeepResearch
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Symbol != "INFY" {
		t.Errorf("expected normalised symbol INFY, got %q", result.Symbol)
	}
}

func TestHandleDeepResearch_EmptyReports(t *testing.T) {
	svc := NewService(&mockProvider{name: "BSE", reports: []AnnualReport{}})
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/stock/WIPRO/deep-research", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result DeepResearch
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result.AnnualReports) != 0 {
		t.Errorf("expected 0 reports, got %d", len(result.AnnualReports))
	}
}

func TestHandleDeepResearch_FallbackProvider(t *testing.T) {
	// First provider fails, second succeeds — handler returns 200 with BSE data.
	bseReports := []AnnualReport{{SeqNumber: 1, Year: "2024"}}
	svc := NewService(
		&mockProvider{name: "NSE", err: errors.New("NSE down")},
		&mockProvider{name: "BSE", reports: bseReports},
	)
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/stock/SBIN/deep-research", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result DeepResearch
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.AnnualReportsSource != "BSE" {
		t.Errorf("expected source BSE, got %q", result.AnnualReportsSource)
	}
}
