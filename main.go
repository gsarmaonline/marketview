package main

import (
	"log"

	"marketview/internal/api"
	"marketview/internal/deepresearch"
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

	drService := deepresearch.NewService(nseClient)
	drHandler := deepresearch.NewHandler(drService)

	srv := api.New(allIndicators, mfHandler, drHandler)
	log.Fatal(srv.Run(":8080"))
}
