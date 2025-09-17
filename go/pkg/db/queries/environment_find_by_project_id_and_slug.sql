-- name: FindEnvironmentByProjectIdAndSlug :one
SELECT id, workspace_id, project_id, slug, description
FROM environments
WHERE workspace_id = sqlc.arg(workspace_id) 
  AND project_id = sqlc.arg(project_id) 
  AND slug = sqlc.arg(slug);
