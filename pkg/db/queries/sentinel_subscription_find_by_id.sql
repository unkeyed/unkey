-- name: FindSentinelSubscriptionByID :one
SELECT * FROM sentinel_subscriptions
WHERE id = sqlc.arg(id)
LIMIT 1;
