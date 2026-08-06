-- name: InsertPortal :exec
INSERT INTO portals (
    id,
    workspace_id,
    slug,
    app_id,
    keyspace_id,
    enabled,
    return_url,
    logo_url,
    primary_color,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(slug),
    sqlc.narg(app_id),
    sqlc.narg(keyspace_id),
    sqlc.arg(enabled),
    sqlc.narg(return_url),
    sqlc.narg(logo_url),
    sqlc.narg(primary_color),
    sqlc.arg(created_at),
    sqlc.narg(updated_at)
);
