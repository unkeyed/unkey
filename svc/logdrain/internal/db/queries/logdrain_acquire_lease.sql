-- AcquireLogdrainLease assigns a new fencing token and computes an absolute
-- expiry from database time and the supplied TTL. The update rechecks both
-- expiry and enabled state so competing lease services need no transaction.
-- name: AcquireLogdrainLease :execrows
UPDATE logdrain_state s
SET s.lease_id = sqlc.arg(lease_id),
  s.fencing_token = sqlc.arg(fencing_token),
  s.lease_expires_at = CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED) + CAST(sqlc.arg(ttl_millis) AS SIGNED)
WHERE s.logdrain_id = sqlc.arg(logdrain_id)
  AND s.lease_expires_at <= CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
  AND EXISTS (
    SELECT 1 FROM logdrains d
    WHERE d.id = s.logdrain_id AND d.enabled = true
  );
