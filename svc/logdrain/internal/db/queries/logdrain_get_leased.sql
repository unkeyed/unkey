-- GetLeasedLogdrain returns one due drain only while the caller still owns its
-- database-time lease.
-- The caller must use the same fencing token for every later state mutation.
-- name: GetLeasedLogdrain :one
SELECT
  d.id,
  d.workspace_id,
  d.stream,
  d.config,
  s.consecutive_failures,
  s.committed_offset_inserted_at,
  s.committed_offset_event_id,
  s.fencing_token
FROM logdrains d
JOIN logdrain_state s ON s.logdrain_id = d.id
WHERE d.id = sqlc.arg(logdrain_id)
  AND d.enabled = true
  AND s.status = 'active'
  AND s.fencing_token = sqlc.arg(fencing_token)
  AND s.lease_expires_at > CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
  AND s.next_attempt_at <= CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED);
