-- name: FindEnvironmentById :one
SELECT id, workspace_id, project_id, app_id, slug, description, current_deployment_id, is_rolled_back
FROM environments
WHERE id = sqlc.arg(id);
