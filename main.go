package main

import (
	"log"
	"net/http"

	"marketview/internal/api"
	"marketview/internal/indicators"
	"marketview/internal/mutualfund"
	"marketview/internal/nse"
)

func main() {
	nseClient, err := nse.New()
	if err != nil {
		log.Fatalf("failed to initialise NSE client: %v", err)
	}

	allIndicators := []indicators.Indicator{
		indicators.NewNiftyPE(nseClient),
	}

	mfService := mutualfund.NewService()
	mfHandler := mutualfund.NewHandler(mfService)

	srv := api.New(allIndicators, mfHandler)
	srv.RegisterRoutes()

	log.Println("server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
