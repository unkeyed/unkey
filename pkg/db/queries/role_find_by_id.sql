-- name: FindRoleByID :one
-- Finds a role record by its ID
-- Returns: The role record if found
SELECT roles.pk, roles.id, roles.workspace_id, roles.project_id, roles.name, roles.description, roles.created_at_m, roles.updated_at_m
FROM roles
WHERE id = sqlc.arg(role_id)
LIMIT 1;
