-- name: LockDeploymentsByEnvironmentId :many
-- LockDeploymentsByEnvironmentId serializes environment cleanup with status
-- reports. Hold these locks until instances and deployments have been deleted.
SELECT id FROM deployments
WHERE environment_id = sqlc.arg(environment_id)
ORDER BY id
FOR UPDATE;
