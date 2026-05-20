-- name: ListActiveDeploymentsByProjectId :many
-- Returns deployments still in a non-terminal status. The project delete
-- workflow uses this to cancel in-flight Restate invocations before the
-- cascade drops their rows; the NOT IN list mirrors the terminal set in
-- deployment_update_status_if_active.sql so the two queries stay aligned.
SELECT id, invocation_id
FROM deployments
WHERE project_id = sqlc.arg('project_id')
  AND status NOT IN ('ready', 'failed', 'skipped', 'stopped', 'superseded', 'cancelled');
