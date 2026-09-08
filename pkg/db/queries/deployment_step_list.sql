-- name: ListFailedDeploymentStepsByIds :many
SELECT deployment_steps.pk, deployment_steps.workspace_id, deployment_steps.project_id, deployment_steps.environment_id, deployment_steps.deployment_id, deployment_steps.app_id, deployment_steps.step, deployment_steps.started_at, deployment_steps.ended_at, deployment_steps.error FROM deployment_steps
WHERE workspace_id = sqlc.arg(workspace_id)
  AND deployment_id IN (sqlc.slice('deployment_ids'))
  AND error IS NOT NULL AND error != ''
ORDER BY deployment_id, started_at ASC;
