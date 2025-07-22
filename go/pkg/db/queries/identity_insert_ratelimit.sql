-- name: InsertIdentityRatelimit :exec
INSERT INTO `ratelimits` (
    id,
    workspace_id,
    identity_id,
    name,
    `limit`,
    duration,
    created_at,
    auto_apply
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('workspace_id'),
    sqlc.arg('identity_id'),
    sqlc.arg('name'),
    sqlc.arg('limit'),
    sqlc.arg('duration'),
    sqlc.arg('created_at'),
    sqlc.arg('auto_apply')
) ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    `limit` = VALUES(`limit`),
    duration = VALUES(duration),
    auto_apply = VALUES(auto_apply),
    updated_at = VALUES(created_at);
