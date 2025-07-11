-- name: InsertVersionStep :exec
INSERT INTO version_steps (
    version_id,
    status,
    message,
    error_message,
    created_at
) VALUES (
    ?, ?, ?, ?, ?
)
ON DUPLICATE KEY UPDATE
    message = VALUES(message),
    error_message = VALUES(error_message),
    created_at = VALUES(created_at);