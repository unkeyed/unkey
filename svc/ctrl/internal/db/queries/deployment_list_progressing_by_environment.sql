-- name: ListProgressingDeploymentsByEnvironmentId :many
-- Returns deployments in a non-terminal (progressing) status for an
-- environment. The environment delete workflow uses this to cancel
-- in-flight Restate invocations before the cascade drops deployment
-- rows. Callers pass db.ProgressingDeploymentStatuses so the
-- progressing set has a single source of truth; new statuses default
-- to NOT progressing, so unknown states are skipped rather than
-- accidentally cancelled.
SELECT id, invocation_id
FROM deployments
WHERE environment_id = sqlc.arg('environment_id')
  AND status IN (sqlc.slice('progressing_statuses'));
