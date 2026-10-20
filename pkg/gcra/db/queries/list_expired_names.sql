-- name: ListExpiredNames :many
-- ListExpiredNames returns names with at least one row that expired before the
-- cutoff. The caller must still check every row of a name before deleting it.
SELECT DISTINCT name
FROM ratelimit_gcra
WHERE expires_at_ns < sqlc.arg(cutoff_ns)
LIMIT ?;
