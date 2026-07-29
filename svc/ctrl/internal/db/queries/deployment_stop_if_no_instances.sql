-- name: StopDeploymentIfNoInstances :exec
-- StopDeploymentIfNoInstances finalizes a requested stop only after krane has
-- reported that no instances remain. The desired_state guard prevents stale
-- delete reports from marking a deployment stopped after it has been woken.
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
UPDATE deployments d
LEFT JOIN instances i ON (i.deployment_id = d.id COLLATE utf8mb4_0900_ai_ci AND i.deployment_id = d.id COLLATE utf8mb4_0900_as_cs)
SET d.status = 'stopped', d.updated_at = sqlc.arg(updated_at)
WHERE d.id = sqlc.arg(id)
  AND d.desired_state = 'stopped'
  AND i.deployment_id IS NULL;
