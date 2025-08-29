-- name: Set :exec
INSERT INTO kv (`key`, workspace_id, value, ttl, created_at)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    value = VALUES(value),
    ttl = VALUES(ttl),
    created_at = VALUES(created_at);