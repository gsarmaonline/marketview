package main

import (
	"context"
	"log"

	"marketview/internal/api"
	"marketview/internal/db"
	"marketview/internal/deepresearch"
	"marketview/internal/indicators"
	"marketview/internal/mutualfund"
	"marketview/internal/news"
	"marketview/internal/nse"
)

func main() {
	ctx := context.Background()

	pool, err := db.Open(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	nseClient, err := nse.New()
	if err != nil {
		log.Fatalf("failed to initialise NSE client: %v", err)
	}

	allIndicators := []indicators.Indicator{
		indicators.NewNiftyPE(nseClient),
	}

	mfService := mutualfund.NewService()
	mfHandler := mutualfund.NewHandler(mfService)

	drStore := deepresearch.NewStore(pool)
	drService := deepresearch.NewService(drStore,
		deepresearch.NewNSEProvider(nseClient),
		deepresearch.NewBSEProvider(),
	)
	drHandler := deepresearch.NewHandler(drService)

	newsStore := news.NewStore()

	srv, err := api.New(ctx, pool, allIndicators, mfHandler, newsStore, drHandler)
	if err != nil {
		log.Fatalf("failed to initialise server: %v", err)
	}
	defer srv.Shutdown()

	log.Fatal(srv.Run(":8080"))
}
