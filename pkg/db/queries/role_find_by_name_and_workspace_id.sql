-- name: FindRoleByNameAndWorkspaceID :one
-- Finds a role record by its name within a specific workspace
-- Returns: The role record if found
SELECT *
FROM roles
WHERE name = sqlc.arg(name)
AND workspace_id = sqlc.arg(workspace_id)
LIMIT 1;
