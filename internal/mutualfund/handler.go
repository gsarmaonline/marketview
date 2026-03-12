package mutualfund

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// Handler handles mutual fund HTTP requests.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// HandleSearch handles GET /api/mutual-fund/search?q={query}
func (h *Handler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	results, err := h.service.Search(query)
	if err != nil {
		http.Error(w, "search failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, results)
}

// HandleDetails handles GET /api/mutual-fund/{schemeCode}
func (h *Handler) HandleDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract scheme code from the trailing path segment.
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	codeStr := parts[len(parts)-1]
	schemeCode, err := strconv.Atoi(codeStr)
	if err != nil || schemeCode <= 0 {
		http.Error(w, "invalid scheme code", http.StatusBadRequest)
		return
	}

	details, err := h.service.GetDetails(schemeCode)
	if err != nil {
		http.Error(w, "failed to fetch fund details: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, details)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
