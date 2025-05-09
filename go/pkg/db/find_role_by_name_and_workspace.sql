-- name: FindRoleByNameAndWorkspace :one
-- Finds a role record by its name within a specific workspace
-- Returns: The role record if found
SELECT *
FROM "roles"
WHERE "name" = $1
AND "workspaceId" = $2
LIMIT 1;