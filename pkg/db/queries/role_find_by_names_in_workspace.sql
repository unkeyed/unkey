-- name: FindRolesByNamesInWorkspace :many
-- FindRolesByNamesInWorkspace returns roles with the requested names from any
-- project in one workspace. Use it to detect cross-project name conflicts
-- before creating project-scoped roles.
SELECT id, project_id, name FROM roles
WHERE workspace_id = sqlc.arg(workspace_id)
  AND name IN (sqlc.slice('names'));
