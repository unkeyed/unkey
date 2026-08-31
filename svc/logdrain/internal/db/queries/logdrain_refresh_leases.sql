-- RefreshLogdrainLeases replaces the expiry for every valid lease assigned to
-- one startup-unique lease ID. Database-side per-row jitter keeps those leases
-- from expiring together. The query cannot revive an expired lease. Per-drain
-- fencing tokens remain unchanged and continue to guard delivery state writes.
-- name: RefreshLogdrainLeases :execrows
UPDATE logdrains
SET lease_expires_at = CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
  + CAST(sqlc.arg(minimum_ttl_millis) AS SIGNED)
  + FLOOR(RAND() * (CAST(sqlc.arg(ttl_jitter_millis) AS SIGNED) + 1))
WHERE lease_id = sqlc.arg(lease_id)
  AND lease_expires_at > CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED);
