-- name: EndActiveDeploymentStepsWithError :exec
UPDATE `deployment_steps`
SET ended_at = sqlc.arg('ended_at'), error = sqlc.arg('error')
WHERE deployment_id = sqlc.arg('deployment_id') AND ended_at IS NULL;
