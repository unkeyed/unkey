-- name: ListActiveDeploymentsByProjectId :many
-- Returns deployments still in a non-terminal status. The project delete
-- workflow uses this to cancel in-flight Restate invocations before the
-- cascade drops their rows. Callers pass db.TerminalDeploymentStatuses so
-- the terminal set has a single source of truth shared with
-- UpdateDeploymentStatusIfActive.
SELECT id, invocation_id
FROM deployments
WHERE project_id = sqlc.arg('project_id')
  AND status NOT IN (sqlc.slice('terminal_statuses'));
