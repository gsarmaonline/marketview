package deepresearch

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Cache persists parsed supply chain results in Postgres so that the
// expensive Python PDF parsing step is only run once per (symbol, year).
type Cache struct {
	pool *pgxpool.Pool
}

// NewCache creates a Cache backed by the given connection pool.
func NewCache(pool *pgxpool.Pool) *Cache {
	return &Cache{pool: pool}
}

// Get returns cached entities for (symbol, reportYear). The second return
// value is false when no cached entry exists.
func (c *Cache) Get(ctx context.Context, symbol, reportYear string) ([]SupplyChainEntity, bool, error) {
	var raw []byte
	err := c.pool.QueryRow(ctx,
		`SELECT entities FROM supply_chain_cache WHERE symbol=$1 AND report_year=$2`,
		symbol, reportYear,
	).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var entities []SupplyChainEntity
	if err := json.Unmarshal(raw, &entities); err != nil {
		return nil, false, err
	}
	return entities, true, nil
}

// Set stores (or replaces) the parsed entities for (symbol, reportYear).
func (c *Cache) Set(ctx context.Context, symbol, reportYear string, entities []SupplyChainEntity) error {
	raw, err := json.Marshal(entities)
	if err != nil {
		return err
	}
	_, err = c.pool.Exec(ctx,
		`INSERT INTO supply_chain_cache (symbol, report_year, entities)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (symbol, report_year) DO UPDATE SET entities=$3, parsed_at=NOW()`,
		symbol, reportYear, raw,
	)
	return err
}
