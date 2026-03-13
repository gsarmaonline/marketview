-- name: GetLatestShareholding :one
SELECT pattern, fetched_at FROM shareholding_pattern_store
WHERE symbol = $1
ORDER BY fetched_at DESC
LIMIT 1;

-- name: UpsertShareholding :exec
INSERT INTO shareholding_pattern_store (symbol, quarter_end, pattern)
VALUES ($1, $2, $3)
ON CONFLICT (symbol, quarter_end) DO UPDATE
    SET pattern = EXCLUDED.pattern, fetched_at = NOW();
