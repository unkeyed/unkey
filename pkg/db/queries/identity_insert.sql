-- name: InsertIdentity :exec
INSERT INTO `identities` (
    id,
    external_id,
    workspace_id,
    environment,
    created_at,
    meta
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('external_id'),
    sqlc.arg('workspace_id'),
    sqlc.arg('environment'),
    sqlc.arg('created_at'),
    CAST(sqlc.arg('meta') AS JSON)
);
