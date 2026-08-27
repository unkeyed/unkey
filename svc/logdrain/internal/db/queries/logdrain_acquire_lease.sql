-- AcquireLogdrainLease assigns a new fencing token and computes an absolute
-- expiry from database time and the supplied TTL.
-- Dashboard updates lock logdrains before logdrain_state. Lease acquisition
-- must use the same lock order or each transaction can wait for a row held by
-- the other. Call LockEnabledLogdrainsForUpdate in the same transaction first.
-- name: AcquireLogdrainLease :execrows
UPDATE logdrain_state s
SET s.lease_id = sqlc.arg(lease_id),
  s.fencing_token = sqlc.arg(fencing_token),
  s.lease_expires_at = CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED) + CAST(sqlc.arg(ttl_millis) AS SIGNED)
WHERE s.logdrain_id = sqlc.arg(logdrain_id)
  AND s.lease_expires_at <= CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED);
