-- name: TryAcquireBuildSlot :execrows
-- Atomically grants a build slot if the workspace is below its
-- max_concurrent_builds quota. The capacity count joins against
-- deployments.status so any slot whose deployment has gone terminal
-- (compensation never ran, invocation purged, etc.) is automatically
-- excluded from the count — leaked rows are inert, not deadlocks.
-- Returns 1 row inserted on grant, 0 when at capacity.
INSERT INTO build_slots (deployment_id, workspace_id, acquired_at)
SELECT
  sqlc.arg('deployment_id'),
  sqlc.arg('workspace_id'),
  sqlc.arg('acquired_at')
WHERE (
  SELECT COUNT(*) FROM build_slots bs
  JOIN deployments d ON d.id = bs.deployment_id
  WHERE bs.workspace_id = sqlc.arg('workspace_id')
    AND d.status NOT IN (sqlc.slice('terminal_statuses'))
) < (
  SELECT q.max_concurrent_builds FROM quota q
  WHERE q.workspace_id = sqlc.arg('workspace_id')
);
