-- name: ListFailedDeploymentStepsByIds :many
SELECT * FROM deployment_steps
WHERE workspace_id = sqlc.arg(workspace_id)
  AND deployment_id IN (sqlc.slice('deployment_ids'))
  AND error IS NOT NULL AND error != ''
ORDER BY deployment_id, started_at ASC;
