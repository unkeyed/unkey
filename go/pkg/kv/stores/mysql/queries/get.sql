-- name: Get :one
SELECT * FROM kv
WHERE `key` = ? AND (ttl IS NULL OR ttl > ?);