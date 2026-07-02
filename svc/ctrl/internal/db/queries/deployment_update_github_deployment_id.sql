-- name: UpdateDeploymentGithubDeploymentId :exec
UPDATE deployments
SET github_deployment_id = sqlc.arg(github_deployment_id), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
