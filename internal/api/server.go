package api

import (
	"encoding/json"
	"log"
	"net/http"

	"marketview/internal/indicators"
	"marketview/internal/mutualfund"
)

// Server wires up all HTTP routes and their dependencies.
type Server struct {
	indicators []indicators.Indicator
	mfHandler  *mutualfund.Handler
}

func New(inds []indicators.Indicator, mfHandler *mutualfund.Handler) *Server {
	return &Server{indicators: inds, mfHandler: mfHandler}
}

// RegisterRoutes registers all application routes on the default ServeMux.
func (s *Server) RegisterRoutes() {
	http.HandleFunc("/api/indicators", s.withCORS(s.handleIndicators))
	http.HandleFunc("/api/mutual-fund/search", s.withCORS(s.mfHandler.HandleSearch))
	http.HandleFunc("/api/mutual-fund/", s.withCORS(s.mfHandler.HandleDetails))
}

func (s *Server) handleIndicators(w http.ResponseWriter, r *http.Request) {
	results := make([]indicators.IndicatorResult, 0, len(s.indicators))
	for _, ind := range s.indicators {
		result, err := ind.Fetch()
		if err != nil {
			log.Printf("error fetching %s: %v", ind.Name(), err)
			continue
		}
		results = append(results, result)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *Server) withCORS(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}
