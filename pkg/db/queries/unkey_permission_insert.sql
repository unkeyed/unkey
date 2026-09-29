-- name: InsertUnkeyPermission :exec
-- InsertUnkeyPermission assigns a permission directly to a principal in the
-- workspace it authorizes. Duplicate permissions for that principal are rejected.
INSERT INTO unkey_permissions (
    id,
    for_workspace_id,
    principal_type,
    principal_id,
    slug,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(for_workspace_id),
    sqlc.arg(principal_type),
    sqlc.arg(principal_id),
    sqlc.arg(slug),
    sqlc.arg(created_at)
);
