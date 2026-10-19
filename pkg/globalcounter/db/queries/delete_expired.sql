-- name: DeleteExpired :execrows
-- DeleteExpired bounds cleanup work and removes the oldest eligible rows first.
-- The caller chooses a safe cutoff; rows without a delete_after are retained.
DELETE FROM counters
WHERE delete_after <= sqlc.arg(cutoff_ms)
ORDER BY delete_after
LIMIT ?;
