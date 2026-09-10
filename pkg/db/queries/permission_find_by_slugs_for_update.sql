-- name: FindPermissionsBySlugsForUpdate :many
-- FindPermissionsBySlugsForUpdate locks matching permissions in one project so
-- role assignment cannot attach permissions owned by another project.
SELECT id, name, slug, description
FROM permissions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND project_id = sqlc.arg(project_id)
  AND slug IN (sqlc.slice('slugs'))
ORDER BY slug
FOR UPDATE;
