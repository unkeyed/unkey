-- name: CountSentinelsByAppId :one
SELECT COUNT(*) as count
FROM sentinels s
JOIN environments e ON s.environment_id = e.id
WHERE e.app_id = sqlc.arg(app_id);
