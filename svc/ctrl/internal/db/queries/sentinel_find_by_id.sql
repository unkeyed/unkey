-- name: FindSentinelByID :one
SELECT * FROM sentinels s
WHERE id = sqlc.arg(id) LIMIT 1;
