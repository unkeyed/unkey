-- name: FindEnvironmentByProjectIdAndSlug :one
SELECT *
FROM environments
WHERE workspace_id = sqlc.arg(workspace_id)
  AND project_id = sqlc.arg(project_id)
  AND slug = sqlc.arg(slug);
