-- name: FindEnvironmentByWorkspaceAndSlug :one
SELECT id, workspace_id, project_id, slug, description
FROM environments
WHERE workspace_id = sqlc.arg(workspace_id) AND slug = sqlc.arg(slug);
