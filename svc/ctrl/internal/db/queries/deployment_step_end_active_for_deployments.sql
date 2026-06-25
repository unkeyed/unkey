-- name: EndActiveDeploymentStepsForDeployments :exec
UPDATE `deployment_steps`
SET ended_at = sqlc.arg('ended_at'), error = sqlc.arg('error')
WHERE deployment_id IN (sqlc.slice('deployment_ids')) AND ended_at IS NULL;
