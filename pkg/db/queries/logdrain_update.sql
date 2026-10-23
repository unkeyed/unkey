-- name: UpdateLogdrain :exec
-- Caller holds the drain lock. Keep the committed cursor unchanged and expire
-- leases for delivery changes so in-flight workers cannot commit stale state.
UPDATE logdrains SET name = sqlc.arg(name), config = sqlc.arg(config), status = sqlc.arg(status),
  lease_expires_at = sqlc.arg(lease_expires_at), consecutive_failures = sqlc.arg(consecutive_failures),
  next_attempt_at = sqlc.arg(next_attempt_at), updated_at = sqlc.arg(updated_at)
WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id);
