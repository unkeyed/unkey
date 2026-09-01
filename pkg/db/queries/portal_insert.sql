-- name: InsertPortal :exec
INSERT INTO portals (
    id,
    workspace_id,
    slug,
    display_name,
    app_id,
    key_auth_id,
    enabled,
    logo_url,
    primary_color,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(slug),
    sqlc.arg(display_name),
    sqlc.narg(app_id),
    sqlc.narg(key_auth_id),
    sqlc.arg(enabled),
    sqlc.narg(logo_url),
    sqlc.narg(primary_color),
    sqlc.arg(created_at),
    sqlc.narg(updated_at)
);
