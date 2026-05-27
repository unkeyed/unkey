-- name: PreGrantBuildSlot :execrows
-- Pre-grants a slot to a promoted waiter, but only if the waiter's
-- deployment hasn't already gone terminal. 0 rows means the waiter is
-- defunct and the caller should skip it and try the next one.
INSERT INTO build_slots (deployment_id, workspace_id, acquired_at)
SELECT
  sqlc.arg('deployment_id'),
  sqlc.arg('workspace_id'),
  sqlc.arg('acquired_at')
WHERE EXISTS (
  SELECT 1 FROM deployments d
  WHERE d.id = sqlc.arg('deployment_id')
    AND d.status NOT IN (sqlc.slice('terminal_statuses'))
);
