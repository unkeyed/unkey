-- name: FindPermissionsBySlugs :many
-- FindPermissionsBySlugs returns permissions with the requested slugs from one
-- project. The project filter prevents cross-project key assignments.
SELECT * FROM permissions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND project_id = sqlc.arg(project_id)
  AND slug IN (sqlc.slice('slugs'));
