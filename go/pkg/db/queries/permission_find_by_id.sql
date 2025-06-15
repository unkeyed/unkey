-- name: FindPermissionByID :one
-- Finds a permission record by its ID
-- Returns: The permission record if found
SELECT *
FROM permissions
WHERE id = sqlc.arg(permission_id)
LIMIT 1;
