-- name: FindSentinelsByEnvironmentID :many
SELECT sqlc.embed(s), sqlc.embed(sub), sqlc.embed(r)
FROM sentinels s
INNER JOIN sentinel_subscriptions sub ON sub.id = s.subscription_id
LEFT JOIN regions r ON s.region_id = r.id
WHERE s.environment_id = sqlc.arg(environment_id);
