-- name: ClearAppCurrentDeployment :exec
-- Clears apps.current_deployment_id when it still points at the given
-- deployment. Teardown calls this before stopping an app's current deployment so
-- the DeploymentService current-deployment guard permits the change; gating on
-- the deployment id makes it a safe no-op if a concurrent deploy already
-- re-pointed current_deployment_id at something else.
UPDATE apps
SET current_deployment_id = NULL,
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(app_id)
  AND current_deployment_id = sqlc.arg(deployment_id);
