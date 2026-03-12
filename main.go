package main

import (
	"context"
	"log"

	"marketview/internal/api"
	"marketview/internal/db"
	"marketview/internal/indicators"
	"marketview/internal/mutualfund"
	"marketview/internal/news"
	"marketview/internal/nse"
)

func main() {
	ctx := context.Background()

	// Database
	pool, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Println("database ready")

	// NSE client & indicators
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

	srv := api.New(allIndicators, mfHandler, newsStore, pool)
	log.Fatal(srv.Run(":8080"))
}
