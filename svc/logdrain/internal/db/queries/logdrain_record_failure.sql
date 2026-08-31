-- RecordLogdrainFailure atomically increments failures and optionally pauses
-- the drain. Database time computes the absolute retry time. The update requires
-- the exact fencing token, a valid lease, and a running drain. The status guard
-- prevents an in-flight failure from overriding a user pause.
-- name: RecordLogdrainFailure :execrows
UPDATE logdrains
SET consecutive_failures = consecutive_failures + 1,
  status = sqlc.arg(status),
  next_attempt_at = CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED) + CAST(sqlc.arg(retry_after_millis) AS SIGNED)
WHERE id = sqlc.arg(logdrain_id)
  AND status = 'running'
  AND fencing_token = sqlc.arg(fencing_token)
  AND lease_expires_at > CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED);
