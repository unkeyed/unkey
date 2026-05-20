-- name: ListActiveDeploymentsByEnvironmentId :many
-- Returns deployments still in a non-terminal status for an environment.
-- The environment delete workflow uses this to cancel in-flight Restate
-- invocations before the cascade drops deployment rows. Callers pass
-- db.TerminalDeploymentStatuses so the terminal set has a single source
-- of truth shared with UpdateDeploymentStatusIfActive.
SELECT id, invocation_id
FROM deployments
WHERE environment_id = sqlc.arg('environment_id')
  AND status NOT IN (sqlc.slice('terminal_statuses'));
