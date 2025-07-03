-- name: InsertKeyRatelimit :exec
INSERT INTO `ratelimits` (
    id,
    workspace_id,
    key_id,
    name,
    `limit`,
    duration,
    auto_apply,
    created_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('workspace_id'),
    sqlc.arg('key_id'),
    sqlc.arg('name'),
    sqlc.arg('limit'),
    sqlc.arg('duration'),
    sqlc.arg('auto_apply'),
    sqlc.arg('created_at')
);
