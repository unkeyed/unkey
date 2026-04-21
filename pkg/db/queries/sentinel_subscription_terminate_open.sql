-- name: TerminateOpenSentinelSubscription :exec
-- Closes the single open subscription row for a sentinel. Paired with
-- InsertSentinelSubscription to rotate tier / replica eras in one tx; the
-- `one_open_subscription_per_sentinel` unique constraint would reject a
-- second open row if the terminate step is skipped.
UPDATE sentinel_subscriptions
SET terminated_at = sqlc.arg(terminated_at)
WHERE sentinel_id = sqlc.arg(sentinel_id)
  AND terminated_at IS NULL;
