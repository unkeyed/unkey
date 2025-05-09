-- name: FindPermissionByNameAndWorkspace :one
-- Finds a permission record by its name within a specific workspace
-- Returns: The permission record if found
SELECT *
FROM "permissions"
WHERE "name" = $1
AND "workspaceId" = $2
LIMIT 1;