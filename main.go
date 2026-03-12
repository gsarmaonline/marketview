package main

import (
	"encoding/json"
	"log"
	"marketview/internal/indicators"
	"marketview/internal/nse"
	"net/http"
)

func main() {
	nseClient, err := nse.New()
	if err != nil {
		log.Fatalf("failed to initialise NSE client: %v", err)
	}

	allIndicators := []indicators.Indicator{
		indicators.NewNiftyPE(nseClient),
	}

	http.HandleFunc("/api/indicators", func(w http.ResponseWriter, r *http.Request) {
		results := make([]indicators.IndicatorResult, 0, len(allIndicators))

		for _, ind := range allIndicators {
			result, err := ind.Fetch()
			if err != nil {
				log.Printf("error fetching %s: %v", ind.Name(), err)
				continue
			}
			results = append(results, result)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		json.NewEncoder(w).Encode(results)
	})

	log.Println("server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
