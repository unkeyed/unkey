-- UpdateLogdrainNextAttemptAt sets the retry schedule and failure count, for
-- example to skip the production retry backoff. Only tests call it.
-- name: UpdateLogdrainNextAttemptAt :exec
UPDATE logdrains
SET next_attempt_at = sqlc.arg(next_attempt_at),
  consecutive_failures = sqlc.arg(consecutive_failures)
WHERE id = sqlc.arg(id);
