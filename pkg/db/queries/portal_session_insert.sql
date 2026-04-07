-- name: InsertPortalSession :exec
INSERT INTO portal_sessions (
    id,
    workspace_id,
    portal_config_id,
    external_id,
    metadata,
    permissions,
    preview,
    expires_at,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(portal_config_id),
    sqlc.arg(external_id),
    sqlc.narg(metadata),
    sqlc.arg(permissions),
    sqlc.arg(preview),
    sqlc.arg(expires_at),
    sqlc.arg(created_at)
);
