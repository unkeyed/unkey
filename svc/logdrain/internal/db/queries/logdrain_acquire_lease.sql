-- AcquireLogdrainLease assigns a new fencing token and computes an absolute
-- expiry from database time and the supplied TTL. The update rechecks both
-- expiry and running state so competing lease services need no transaction.
-- name: AcquireLogdrainLease :execrows
UPDATE logdrains
SET lease_id = sqlc.arg(lease_id),
  fencing_token = sqlc.arg(fencing_token),
  lease_expires_at = CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED) + CAST(sqlc.arg(ttl_millis) AS SIGNED)
WHERE id = sqlc.arg(logdrain_id)
  AND status = 'running'
  AND lease_expires_at <= CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED);
