-- name: ListHoldings :many
SELECT id, asset_type, name, quantity, buy_price, current_value,
       buy_date, notes, metadata, created_at, updated_at
FROM holdings
ORDER BY asset_type, created_at DESC;

-- name: CreateHolding :one
INSERT INTO holdings (asset_type, name, quantity, buy_price, current_value, buy_date, notes, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, asset_type, name, quantity, buy_price, current_value,
          buy_date, notes, metadata, created_at, updated_at;

-- name: UpdateHolding :one
UPDATE holdings
SET asset_type    = $1,
    name          = $2,
    quantity      = $3,
    buy_price     = $4,
    current_value = $5,
    buy_date      = $6,
    notes         = $7,
    metadata      = $8,
    updated_at    = NOW()
WHERE id = $9
RETURNING id, asset_type, name, quantity, buy_price, current_value,
          buy_date, notes, metadata, created_at, updated_at;

-- name: DeleteHolding :execrows
DELETE FROM holdings WHERE id = $1;
