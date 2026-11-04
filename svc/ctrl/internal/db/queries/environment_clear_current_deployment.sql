-- name: ClearEnvironmentCurrentDeployment :exec
-- Gated on the deployment id so it is a no-op if a concurrent deploy already
-- re-pointed current_deployment_id at something else.
UPDATE environments
SET current_deployment_id = NULL,
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(environment_id)
  AND current_deployment_id = sqlc.arg(deployment_id);
