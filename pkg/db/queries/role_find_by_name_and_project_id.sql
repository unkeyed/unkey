-- name: FindRoleByNameAndProjectID :one
-- FindRoleByNameAndProjectID resolves a name in one project while the workspace
-- predicate preserves tenant isolation.
SELECT roles.pk, roles.id, roles.workspace_id, roles.project_id, roles.name, roles.description, roles.created_at_m, roles.updated_at_m
FROM roles
WHERE name = sqlc.arg(name)
AND project_id = sqlc.arg(project_id)
AND workspace_id = sqlc.arg(workspace_id)
LIMIT 1;
