package mutualfund

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestRouter(h *Handler) *gin.Engine {
	r := gin.New()
	r.GET("/api/mutual-fund/search", h.HandleSearch)
	r.GET("/api/mutual-fund/:schemeCode", h.HandleDetails)
	return r
}

func TestHandleSearch_MissingQuery(t *testing.T) {
	h := NewHandler(NewService())
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/mutual-fund/search", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing query, got %d", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body["error"] == "" {
		t.Error("expected non-empty error message in response")
	}
}

func TestHandleDetails_NonNumericSchemeCode(t *testing.T) {
	h := NewHandler(NewService())
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/mutual-fund/notanumber", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-numeric scheme code, got %d", w.Code)
	}
}

func TestHandleDetails_ZeroSchemeCode(t *testing.T) {
	h := NewHandler(NewService())
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/mutual-fund/0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for scheme code 0, got %d", w.Code)
	}
}

func TestHandleDetails_NegativeSchemeCode(t *testing.T) {
	h := NewHandler(NewService())
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/mutual-fund/-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for negative scheme code, got %d", w.Code)
	}
}
