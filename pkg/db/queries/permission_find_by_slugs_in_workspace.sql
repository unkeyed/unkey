-- name: FindPermissionsBySlugsInWorkspace :many
-- FindPermissionsBySlugsInWorkspace returns permissions with the requested
-- slugs from any project in one workspace. Use it to detect cross-project slug
-- conflicts before creating project-scoped permissions.
SELECT * FROM permissions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND slug IN (sqlc.slice('slugs'));
