-- name: GetSupplyChain :one
SELECT entities FROM supply_chain_store
WHERE symbol = $1 AND report_year = $2;

-- name: UpsertSupplyChain :exec
INSERT INTO supply_chain_store (symbol, report_year, entities)
VALUES ($1, $2, $3)
ON CONFLICT (symbol, report_year) DO UPDATE
    SET entities = EXCLUDED.entities, parsed_at = NOW();
