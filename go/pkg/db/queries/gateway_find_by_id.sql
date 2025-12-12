-- name: FindSentinelByID :one
SELECT * FROM sentinels WHERE id = sqlc.arg(id) LIMIT 1;
