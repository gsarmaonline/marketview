package main

import (
	"context"
	"log"
	"os"
	"time"

	"marketview/internal/api"
	"marketview/internal/db"
	"marketview/internal/deepresearch"
	"marketview/internal/indicators"
	"marketview/internal/mutualfund"
	"marketview/internal/news"
	"marketview/internal/nse"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	ingester := news.NewIngester(newsStore, news.NiftyStocks, 15*time.Minute)
	ingester.Start(ctx)

	srv, err := api.New(ctx, pool, allIndicators, mfHandler, newsStore, drHandler)
	if err != nil {
		log.Fatalf("failed to initialise server: %v", err)
	}
	defer srv.Shutdown()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(srv.Run(":" + port))
}
