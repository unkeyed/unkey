-- name: InsertUnkeyPermission :exec
-- InsertUnkeyPermission assigns a permission directly to a principal in the
-- customer workspace that owns it. Duplicate permissions for that principal are rejected.
INSERT INTO unkey_principal_permissions (
    id,
    workspace_id,
    principal_type,
    principal_id,
    slug,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(principal_type),
    sqlc.arg(principal_id),
    sqlc.arg(slug),
    sqlc.arg(created_at)
);
