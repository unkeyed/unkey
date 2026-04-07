-- name: UpsertPortalBranding :exec
INSERT INTO portal_branding (
    portal_config_id,
    logo_url,
    primary_color,
    secondary_color,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(portal_config_id),
    sqlc.narg(logo_url),
    sqlc.narg(primary_color),
    sqlc.narg(secondary_color),
    sqlc.arg(created_at),
    sqlc.narg(updated_at)
)
ON DUPLICATE KEY UPDATE
    logo_url = VALUES(logo_url),
    primary_color = VALUES(primary_color),
    secondary_color = VALUES(secondary_color),
    updated_at = VALUES(updated_at);
