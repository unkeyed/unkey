-- name: FindPermissionById :one
-- Finds a permission record by its ID
-- Returns: The permission record if found
SELECT *
FROM "permissions"
WHERE "id" = $1
LIMIT 1;