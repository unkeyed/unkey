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
);
