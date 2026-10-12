-- name: FindDeploymentStepByDeploymentAndStep :one
-- FindDeploymentStepByDeploymentAndStep returns one step of a deployment.
-- Only tests use this query.
SELECT pk, workspace_id, project_id, environment_id, deployment_id, app_id, step, started_at, ended_at, error
FROM `deployment_steps`
WHERE deployment_id = sqlc.arg(deployment_id) AND step = sqlc.arg(step);
