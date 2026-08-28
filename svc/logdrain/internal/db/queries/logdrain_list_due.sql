-- ListDueLogdrains returns active, due leases assigned to one service process.
-- Database time controls both lease validity and retry scheduling.
-- name: ListDueLogdrains :many
SELECT id AS logdrain_id, fencing_token
FROM logdrains
WHERE enabled = true
  AND status = 'active'
  AND lease_id = sqlc.arg(lease_id)
  AND lease_expires_at > CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
  AND next_attempt_at <= CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
ORDER BY id;
