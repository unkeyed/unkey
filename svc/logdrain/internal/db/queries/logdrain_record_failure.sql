-- RecordLogdrainFailure atomically increments failures and optionally pauses
-- the drain. Database time computes the absolute retry time. The update requires
-- the exact fencing token and a lease that is valid at database time.
-- name: RecordLogdrainFailure :execrows
UPDATE logdrain_state
SET consecutive_failures = consecutive_failures + 1,
  status = sqlc.arg(status),
  next_attempt_at = CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED) + CAST(sqlc.arg(retry_after_millis) AS SIGNED),
  last_error = sqlc.arg(last_error)
WHERE logdrain_id = sqlc.arg(logdrain_id)
  AND fencing_token = sqlc.arg(fencing_token)
  AND lease_expires_at > CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED);
