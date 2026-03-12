package portfolio

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context) ([]Holding, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, asset_type, name, quantity, buy_price, current_value,
		       buy_date, notes, metadata, created_at, updated_at
		FROM holdings
		ORDER BY asset_type, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holdings []Holding
	for rows.Next() {
		var h Holding
		var meta []byte
		err := rows.Scan(
			&h.ID, &h.AssetType, &h.Name, &h.Quantity, &h.BuyPrice,
			&h.CurrentValue, &h.BuyDate, &h.Notes, &meta,
			&h.CreatedAt, &h.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		h.Metadata = json.RawMessage(meta)
		holdings = append(holdings, h)
	}
	if holdings == nil {
		holdings = []Holding{}
	}
	return holdings, rows.Err()
}

func (r *Repository) Create(ctx context.Context, req CreateHoldingRequest) (Holding, error) {
	meta := req.Metadata
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}

	var h Holding
	var rawMeta []byte
	err := r.pool.QueryRow(ctx, `
		INSERT INTO holdings (asset_type, name, quantity, buy_price, current_value, buy_date, notes, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, asset_type, name, quantity, buy_price, current_value,
		          buy_date, notes, metadata, created_at, updated_at
	`, req.AssetType, req.Name, req.Quantity, req.BuyPrice, req.CurrentValue,
		req.BuyDate, req.Notes, meta,
	).Scan(
		&h.ID, &h.AssetType, &h.Name, &h.Quantity, &h.BuyPrice,
		&h.CurrentValue, &h.BuyDate, &h.Notes, &rawMeta,
		&h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		return Holding{}, err
	}
	h.Metadata = json.RawMessage(rawMeta)
	return h, nil
}

func (r *Repository) Update(ctx context.Context, id int, req UpdateHoldingRequest) (Holding, error) {
	meta := req.Metadata
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}

	var h Holding
	var rawMeta []byte
	err := r.pool.QueryRow(ctx, `
		UPDATE holdings
		SET asset_type = $1, name = $2, quantity = $3, buy_price = $4,
		    current_value = $5, buy_date = $6, notes = $7, metadata = $8,
		    updated_at = NOW()
		WHERE id = $9
		RETURNING id, asset_type, name, quantity, buy_price, current_value,
		          buy_date, notes, metadata, created_at, updated_at
	`, req.AssetType, req.Name, req.Quantity, req.BuyPrice, req.CurrentValue,
		req.BuyDate, req.Notes, meta, id,
	).Scan(
		&h.ID, &h.AssetType, &h.Name, &h.Quantity, &h.BuyPrice,
		&h.CurrentValue, &h.BuyDate, &h.Notes, &rawMeta,
		&h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		return Holding{}, fmt.Errorf("holding %d not found", id)
	}
	h.Metadata = json.RawMessage(rawMeta)
	return h, nil
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM holdings WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("holding %d not found", id)
	}
	return nil
}
