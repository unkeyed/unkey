-- name: TerminateOpenSentinelSubscriptionsByEnvironment :exec
-- Bulk-closes every open subscription for sentinels in the given
-- environment. Called during environment deletion before the sentinels
-- themselves are soft-deleted, so every subscription row has a clean
-- `terminated_at` at deletion time.
UPDATE sentinel_subscriptions sub
INNER JOIN sentinels s ON s.id = sub.sentinel_id
SET sub.terminated_at = sqlc.arg(terminated_at)
WHERE s.environment_id = sqlc.arg(environment_id)
  AND sub.terminated_at IS NULL;
