-- UpdateLogdrainLeaseExpiry sets an absolute lease expiry without changing
-- the owner or fencing token. Only tests call it.
-- name: UpdateLogdrainLeaseExpiry :exec
UPDATE logdrains
SET lease_expires_at = sqlc.arg(lease_expires_at)
WHERE id = sqlc.arg(id);
