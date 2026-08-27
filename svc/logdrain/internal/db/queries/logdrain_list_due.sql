-- ListDueLogdrains returns active, due leases assigned to one service process.
-- Database time controls both lease validity and retry scheduling.
-- name: ListDueLogdrains :many
SELECT s.logdrain_id, s.fencing_token
FROM logdrain_state s
JOIN logdrains d ON d.id = s.logdrain_id
WHERE d.enabled = true
  AND s.status = 'active'
  AND s.lease_id = sqlc.arg(lease_id)
  AND s.lease_expires_at > CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
  AND s.next_attempt_at <= CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
ORDER BY s.logdrain_id;
