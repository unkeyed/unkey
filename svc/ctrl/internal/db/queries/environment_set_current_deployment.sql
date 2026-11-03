-- name: SetEnvironmentCurrentDeployment :exec
-- Guarded on the pointer still being unset, so restoring a suspension record
-- never overwrites a deployment promoted in the meantime.
UPDATE environments
SET current_deployment_id = sqlc.arg(deployment_id),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(environment_id)
  AND current_deployment_id IS NULL;
