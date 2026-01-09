-- name: UpsertIdentity :exec
-- Inserts a new identity or does nothing if one already exists for this workspace/external_id.
-- Use FindIdentityByExternalID after this to get the actual ID.
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
)
ON DUPLICATE KEY UPDATE external_id = external_id;
