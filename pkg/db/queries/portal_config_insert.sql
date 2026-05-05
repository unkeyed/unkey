-- name: InsertPortalConfig :exec
INSERT INTO portal_configurations (
    id,
    workspace_id,
    slug,
    app_id,
    key_auth_id,
    enabled,
    return_url,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(slug),
    sqlc.narg(app_id),
    sqlc.narg(key_auth_id),
    sqlc.arg(enabled),
    sqlc.narg(return_url),
    sqlc.arg(created_at),
    sqlc.narg(updated_at)
);
