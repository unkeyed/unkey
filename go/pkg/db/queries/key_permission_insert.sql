-- name: InsertKeyPermission :exec
INSERT INTO `keys_permissions` (
    key_id,
    permission_id,
    workspace_id,
    created_at_m
) VALUES (
    sqlc.arg(key_id),
    sqlc.arg(permission_id),
    sqlc.arg(workspace_id),
    sqlc.arg(created_at)
) ON DUPLICATE KEY UPDATE updated_at_m = sqlc.arg(updated_at);
