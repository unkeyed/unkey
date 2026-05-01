-- name: WindowCountsDeleteExpired :execrows
-- WindowCountsDeleteExpired removes rows whose grace period has passed and
-- returns the number of rows deleted so the caller can surface it for
-- observability. Called by an external Restate cron, not the ratelimit
-- service itself.
DELETE FROM ratelimit_window_counts
WHERE expires_at < sqlc.arg("cutoff");
