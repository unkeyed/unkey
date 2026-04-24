-- name: ListDeploymentStepsByDeploymentId :many
SELECT * FROM `deployment_steps`
WHERE deployment_id = sqlc.arg(deployment_id)
ORDER BY started_at ASC;
