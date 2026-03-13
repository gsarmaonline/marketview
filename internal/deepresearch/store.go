package deepresearch

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	drdb "marketview/internal/deepresearch/db"
)

// shareholdingCacheTTL controls how long a cached shareholding pattern is
// considered fresh. NSE publishes quarterly filings, so 30 days is safe.
const shareholdingCacheTTL = 30 * 24 * time.Hour

// StoreInterface defines the contract for persisting supply chain and shareholding data.
type StoreInterface interface {
	Get(ctx context.Context, symbol, reportYear string) ([]SupplyChainEntity, bool, error)
	Set(ctx context.Context, symbol, reportYear string, entities []SupplyChainEntity) error
	GetShareholding(ctx context.Context, symbol string) (*ShareholdingPattern, bool, error)
	SetShareholding(ctx context.Context, symbol string, pattern *ShareholdingPattern) error
}

// Store persists parsed supply chain and shareholding results in Postgres.
type Store struct {
	q *drdb.Queries
}

// NewStore creates a Store backed by the given connection pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{q: drdb.New(pool)}
}

// Get returns stored supply chain entities for (symbol, reportYear). The second
// return value is false when no entry exists yet.
func (s *Store) Get(ctx context.Context, symbol, reportYear string) ([]SupplyChainEntity, bool, error) {
	raw, err := s.q.GetSupplyChain(ctx, drdb.GetSupplyChainParams{
		Symbol:     symbol,
		ReportYear: reportYear,
	})
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
	return s.q.UpsertSupplyChain(ctx, drdb.UpsertSupplyChainParams{
		Symbol:     symbol,
		ReportYear: reportYear,
		Entities:   raw,
	})
}

// GetShareholding returns the most recently cached shareholding pattern for
// symbol. Returns (nil, false, nil) on cache miss or when data is stale.
func (s *Store) GetShareholding(ctx context.Context, symbol string) (*ShareholdingPattern, bool, error) {
	row, err := s.q.GetLatestShareholding(ctx, symbol)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if time.Since(row.FetchedAt.Time) > shareholdingCacheTTL {
		return nil, false, nil
	}
	var p ShareholdingPattern
	if err := json.Unmarshal(row.Pattern, &p); err != nil {
		return nil, false, err
	}
	return &p, true, nil
}

// SetShareholding upserts the shareholding pattern for symbol.
func (s *Store) SetShareholding(ctx context.Context, symbol string, pattern *ShareholdingPattern) error {
	raw, err := json.Marshal(pattern)
	if err != nil {
		return err
	}
	return s.q.UpsertShareholding(ctx, drdb.UpsertShareholdingParams{
		Symbol:     symbol,
		QuarterEnd: pattern.QuarterEndDate,
		Pattern:    raw,
	})
}
