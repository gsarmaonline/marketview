package main

import (
	"log"

	"marketview/internal/api"
	"marketview/internal/indicators"
	"marketview/internal/mutualfund"
	"marketview/internal/news"
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

	newsStore := news.NewStore()

	srv := api.New(allIndicators, mfHandler, newsStore)
	log.Fatal(srv.Run(":8080"))
}
