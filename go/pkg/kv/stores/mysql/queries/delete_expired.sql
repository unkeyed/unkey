-- name: DeleteExpired :exec
DELETE FROM kv WHERE `key` = ? AND ttl IS NOT NULL AND ttl <= ?;