-- name: InsertPortalFrontlineRoute :exec
INSERT INTO frontline_routes (
    id,
    route_type,
    portal_config_id,
    path_prefix,
    fully_qualified_domain_name,
    sticky,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(id),
    'portal',
    sqlc.arg(portal_config_id),
    sqlc.arg(path_prefix),
    sqlc.arg(fully_qualified_domain_name),
    'none',
    sqlc.arg(created_at),
    sqlc.narg(updated_at)
);
