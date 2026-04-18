-- name: UpdateSentinelConfig :exec
-- UpdateSentinelConfig updates a sentinel's configuration and deploy status.
-- Used by SentinelService.Deploy() to apply new config before triggering krane.
-- Resource changes (cpu/memory/tier) are performed by inserting a new
-- sentinel_subscriptions row and repointing sentinels.subscription_id via
-- UpdateSentinelSubscription.
UPDATE sentinels SET
  image = sqlc.arg(image),
  desired_replicas = sqlc.arg(desired_replicas),
  deploy_status = sqlc.arg(deploy_status),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
