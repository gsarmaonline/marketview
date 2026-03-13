package portfolio

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestMux creates a mux with a nil repository, sufficient for testing
// validation paths that return before any database call.
func newTestMux() *http.ServeMux {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)
	return mux
}

// ── /api/portfolio/holdings (no ID) ──────────────────────────────────────────

func TestHandleHoldings_MethodNotAllowed(t *testing.T) {
	mux := newTestMux()
	for _, method := range []string{http.MethodPatch, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/api/portfolio/holdings", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /api/portfolio/holdings: got %d, want %d", method, w.Code, http.StatusMethodNotAllowed)
		}
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	mux := newTestMux()
	req := httptest.NewRequest(http.MethodPost, "/api/portfolio/holdings", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestCreate_MissingName(t *testing.T) {
	mux := newTestMux()
	body := `{"asset_type":"stock"}`
	req := httptest.NewRequest(http.MethodPost, "/api/portfolio/holdings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", w.Code)
	}
}

func TestCreate_MissingAssetType(t *testing.T) {
	mux := newTestMux()
	body := `{"name":"RELIANCE"}`
	req := httptest.NewRequest(http.MethodPost, "/api/portfolio/holdings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing asset_type, got %d", w.Code)
	}
}

func TestCreate_MissingBothFields(t *testing.T) {
	mux := newTestMux()
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/portfolio/holdings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d", w.Code)
	}
}

// ── /api/portfolio/holdings/:id ───────────────────────────────────────────────

func TestHandleHoldingByID_NonNumericID(t *testing.T) {
	mux := newTestMux()
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/api/portfolio/holdings/abc", strings.NewReader(`{"name":"x","asset_type":"stock"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s with non-numeric ID: got %d, want %d", method, w.Code, http.StatusBadRequest)
		}
	}
}

func TestHandleHoldingByID_ZeroID(t *testing.T) {
	mux := newTestMux()
	req := httptest.NewRequest(http.MethodPut, "/api/portfolio/holdings/0", strings.NewReader(`{"name":"x","asset_type":"stock"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for id=0, got %d", w.Code)
	}
}

func TestHandleHoldingByID_NegativeID(t *testing.T) {
	mux := newTestMux()
	req := httptest.NewRequest(http.MethodDelete, "/api/portfolio/holdings/-1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for id=-1, got %d", w.Code)
	}
}

func TestHandleHoldingByID_MethodNotAllowed(t *testing.T) {
	mux := newTestMux()
	// GET and POST are not allowed on the /:id route.
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req := httptest.NewRequest(method, "/api/portfolio/holdings/1", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /api/portfolio/holdings/1: got %d, want %d", method, w.Code, http.StatusMethodNotAllowed)
		}
	}
}

func TestUpdate_InvalidJSON(t *testing.T) {
	mux := newTestMux()
	req := httptest.NewRequest(http.MethodPut, "/api/portfolio/holdings/1", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON on update, got %d", w.Code)
	}
}

func TestUpdate_MissingRequiredFields(t *testing.T) {
	mux := newTestMux()
	req := httptest.NewRequest(http.MethodPut, "/api/portfolio/holdings/1", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing required fields on update, got %d", w.Code)
	}
}
