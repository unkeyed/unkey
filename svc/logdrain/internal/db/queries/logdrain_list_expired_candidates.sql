-- ListExpiredLogdrainCandidates returns a bounded set of leases that have
-- expired according to database time.
-- name: ListExpiredLogdrainCandidates :many
SELECT id AS logdrain_id
FROM logdrains
WHERE enabled = true
  AND lease_expires_at <= CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
ORDER BY lease_expires_at, id
LIMIT ?;
