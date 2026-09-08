-- name: FindRoleByNameAndWorkspaceID :one
-- Finds a role record by its name within a specific workspace
-- Returns: The role record if found
SELECT roles.pk, roles.id, roles.workspace_id, roles.project_id, roles.name, roles.description, roles.created_at_m, roles.updated_at_m
FROM roles
WHERE name = sqlc.arg(name)
AND workspace_id = sqlc.arg(workspace_id)
LIMIT 1;
