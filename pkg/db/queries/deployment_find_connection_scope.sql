-- name: FindDeploymentConnectionScope :one
SELECT id, project_id, app_id, environment_id
FROM deployments
WHERE id = sqlc.arg(id) AND workspace_id = sqlc.arg(workspace_id);
