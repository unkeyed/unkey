-- ListExpiredLogdrainCandidates returns a bounded set of leases that have
-- expired according to database time.
-- name: ListExpiredLogdrainCandidates :many
SELECT s.logdrain_id
FROM logdrain_state s
WHERE s.lease_expires_at <= CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
  AND EXISTS (
    SELECT 1 FROM logdrains d
    WHERE d.id = s.logdrain_id AND d.enabled = true
  )
ORDER BY s.lease_expires_at, s.logdrain_id
LIMIT ?;
