package main

import (
	"context"
	"log"

	"marketview/internal/api"
	"marketview/internal/indicators"
	"marketview/internal/mutualfund"
	"marketview/internal/news"
	"marketview/internal/nse"
)

func main() {
	ctx := context.Background()

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

	srv, err := api.New(ctx, allIndicators, mfHandler, newsStore)
	if err != nil {
		log.Fatalf("failed to initialise server: %v", err)
	}
	defer srv.Shutdown()

	log.Fatal(srv.Run(":8080"))
}
