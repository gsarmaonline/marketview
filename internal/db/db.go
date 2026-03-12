package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := getenv("DB_HOST", "localhost")
		port := getenv("DB_PORT", "5432")
		user := getenv("DB_USER", "marketview")
		password := getenv("DB_PASSWORD", "marketview")
		dbname := getenv("DB_NAME", "marketview")
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return pool, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS holdings (
			id           SERIAL PRIMARY KEY,
			asset_type   VARCHAR(50)    NOT NULL,
			name         VARCHAR(255)   NOT NULL,
			quantity     NUMERIC(15,4),
			buy_price    NUMERIC(15,2),
			current_value NUMERIC(15,2),
			buy_date     DATE,
			notes        TEXT           NOT NULL DEFAULT '',
			metadata     JSONB          NOT NULL DEFAULT '{}',
			created_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
			updated_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW()
		);
	`)
	return err
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
