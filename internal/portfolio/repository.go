package portfolio

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbgen "marketview/internal/portfolio/db"
)

type Repository struct {
	q *dbgen.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{q: dbgen.New(pool)}
}

func (r *Repository) List(ctx context.Context) ([]Holding, error) {
	rows, err := r.q.ListHoldings(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Holding, 0, len(rows))
	for _, row := range rows {
		result = append(result, fromDB(row))
	}
	return result, nil
}

func (r *Repository) Create(ctx context.Context, req CreateHoldingRequest) (Holding, error) {
	meta := req.Metadata
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	row, err := r.q.CreateHolding(ctx, dbgen.CreateHoldingParams{
		AssetType:    string(req.AssetType),
		Name:         req.Name,
		Quantity:     toNumeric(req.Quantity),
		BuyPrice:     toNumeric(req.BuyPrice),
		CurrentValue: toNumeric(req.CurrentValue),
		BuyDate:      toDate(req.BuyDate),
		Notes:        req.Notes,
		Metadata:     meta,
	})
	if err != nil {
		return Holding{}, err
	}
	return fromDB(row), nil
}

func (r *Repository) Update(ctx context.Context, id int, req UpdateHoldingRequest) (Holding, error) {
	meta := req.Metadata
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	row, err := r.q.UpdateHolding(ctx, dbgen.UpdateHoldingParams{
		ID:           int32(id),
		AssetType:    string(req.AssetType),
		Name:         req.Name,
		Quantity:     toNumeric(req.Quantity),
		BuyPrice:     toNumeric(req.BuyPrice),
		CurrentValue: toNumeric(req.CurrentValue),
		BuyDate:      toDate(req.BuyDate),
		Notes:        req.Notes,
		Metadata:     meta,
	})
	if err != nil {
		return Holding{}, fmt.Errorf("holding %d not found", id)
	}
	return fromDB(row), nil
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	n, err := r.q.DeleteHolding(ctx, int32(id))
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("holding %d not found", id)
	}
	return nil
}

// ── type conversion helpers ───────────────────────────────────────────────

func fromDB(h dbgen.Holding) Holding {
	return Holding{
		ID:           int(h.ID),
		AssetType:    AssetType(h.AssetType),
		Name:         h.Name,
		Quantity:     fromNumeric(h.Quantity),
		BuyPrice:     fromNumeric(h.BuyPrice),
		CurrentValue: fromNumeric(h.CurrentValue),
		BuyDate:      fromDate(h.BuyDate),
		Notes:        h.Notes,
		Metadata:     h.Metadata,
		CreatedAt:    h.CreatedAt.Time,
		UpdatedAt:    h.UpdatedAt.Time,
	}
}

func toNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	_ = n.Scan(*f)
	return n
}

func fromNumeric(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	f, _ := new(big.Float).SetInt(n.Int).Float64()
	if n.Exp != 0 {
		scale := new(big.Float).SetFloat64(1)
		ten := new(big.Float).SetInt64(10)
		exp := int(n.Exp)
		if exp > 0 {
			for i := 0; i < exp; i++ {
				scale.Mul(scale, ten)
			}
			f, _ = new(big.Float).Mul(new(big.Float).SetFloat64(f), scale).Float64()
		} else {
			for i := 0; i < -exp; i++ {
				scale.Mul(scale, ten)
			}
			f, _ = new(big.Float).Quo(new(big.Float).SetFloat64(f), scale).Float64()
		}
	}
	return &f
}

func toDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func fromDate(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	return &d.Time
}
