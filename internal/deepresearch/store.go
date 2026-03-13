package deepresearch

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persists parsed supply chain results in Postgres so that the
// expensive Python PDF parsing step is only run once per (symbol, year).
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a Store backed by the given connection pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Get returns stored entities for (symbol, reportYear). The second return
// value is false when no entry exists yet.
func (s *Store) Get(ctx context.Context, symbol, reportYear string) ([]SupplyChainEntity, bool, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx,
		`SELECT entities FROM supply_chain_store WHERE symbol=$1 AND report_year=$2`,
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
func (s *Store) Set(ctx context.Context, symbol, reportYear string, entities []SupplyChainEntity) error {
	raw, err := json.Marshal(entities)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO supply_chain_store (symbol, report_year, entities)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (symbol, report_year) DO UPDATE SET entities=$3, parsed_at=NOW()`,
		symbol, reportYear, raw,
	)
	return err
}
