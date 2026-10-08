-- name: CountDeploymentStepsByDeploymentId :one
-- CountDeploymentStepsByDeploymentId counts a deployment's steps. Only tests use this query.
SELECT COUNT(*)
FROM `deployment_steps`
WHERE deployment_id = sqlc.arg(deployment_id);
