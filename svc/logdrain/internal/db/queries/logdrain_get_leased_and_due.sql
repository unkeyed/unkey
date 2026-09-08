-- GetLeasedAndDueLogdrain returns one due drain only while the caller still
-- owns its database-time lease.
-- The caller must use the same fencing token for every later state mutation.
-- name: GetLeasedAndDueLogdrain :one
SELECT
  d.id,
  d.workspace_id,
  d.stream,
  d.config,
  d.consecutive_failures,
  d.committed_offset_inserted_at,
  d.committed_offset_event_id,
  d.fencing_token
FROM logdrains d
WHERE d.id = sqlc.arg(logdrain_id)
  AND d.status = 'running'
  AND d.fencing_token = sqlc.arg(fencing_token)
  AND d.lease_expires_at > CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
  AND d.next_attempt_at <= CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED);
