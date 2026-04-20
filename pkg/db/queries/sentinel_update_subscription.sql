-- name: UpdateSentinelSubscription :exec
-- UpdateSentinelSubscription repoints a sentinel at a new (already-inserted)
-- sentinel_subscriptions row. This is how tier / resource changes take effect:
-- insert a new subscription, call this, let the control-plane watcher roll
-- the pod against the new resource envelope.
UPDATE sentinels SET
  subscription_id = sqlc.arg(subscription_id),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
