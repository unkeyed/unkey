-- UpdateLogdrainConfig replaces a drain's delivery config the way the
-- dashboard does: it expires the lease so in-flight state writes fail and
-- resets failure state. Only tests call it.
-- name: UpdateLogdrainConfig :exec
UPDATE logdrains
SET config = sqlc.arg(config),
  lease_expires_at = 0,
  consecutive_failures = 0,
  next_attempt_at = 0
WHERE id = sqlc.arg(id);
