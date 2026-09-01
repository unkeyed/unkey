-- name: FindPermissionByID :one
-- Finds a permission record by its ID
-- Returns: The permission record if found
SELECT permissions.pk, permissions.id, permissions.workspace_id, permissions.project_id, permissions.name, permissions.slug, permissions.description, permissions.created_at_m, permissions.updated_at_m
FROM permissions
WHERE id = sqlc.arg(permission_id)
LIMIT 1;
