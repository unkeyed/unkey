-- name: FindKeyIDByHash :one
-- FindKeyIDByHash returns just the key ID for a given hash. Use this when
-- you only need the ID for a subsequent mutation and do not need the full
-- verification payload with roles, permissions, and rate limits.
SELECT id FROM `keys` WHERE hash = sqlc.arg(hash) AND deleted_at_m IS NULL;
