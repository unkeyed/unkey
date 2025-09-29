-- name: FindEnvironmentById :one
SELECT id, workspace_id, project_id, slug, description
FROM environments
WHERE id = sqlc.arg(id);
