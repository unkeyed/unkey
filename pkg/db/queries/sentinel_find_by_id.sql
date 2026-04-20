-- name: FindSentinelByID :one
SELECT sqlc.embed(s), sqlc.embed(sub)
FROM sentinels s
INNER JOIN sentinel_subscriptions sub ON sub.id = s.subscription_id
WHERE s.id = sqlc.arg(id)
LIMIT 1;
