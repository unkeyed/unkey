-- name: FindDeploymentForStep :one
SELECT id, workspace_id, project_id, app_id, environment_id, status
FROM deployments
WHERE id = sqlc.arg(id);
