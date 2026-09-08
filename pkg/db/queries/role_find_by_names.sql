-- name: FindRolesByNames :many
-- FindRolesByNames returns roles with the requested names from one project. The
-- project filter prevents cross-project key assignments.
SELECT id, name FROM roles
WHERE workspace_id = sqlc.arg(workspace_id)
  AND project_id = sqlc.arg(project_id)
  AND name IN (sqlc.slice('names'))
