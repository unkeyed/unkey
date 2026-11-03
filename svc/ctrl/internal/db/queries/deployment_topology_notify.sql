-- name: NotifyDeploymentTopology :exec
-- NotifyDeploymentTopology completes legacy regional journal steps without
-- accessing retired intent columns. Readers reload the current deployment intent.
UPDATE deployment_topology dt
SET dt.updated_at = GREATEST(COALESCE(dt.updated_at, 0) + 1, COALESCE(sqlc.narg(updated_at), 0))
WHERE dt.deployment_id = sqlc.arg(deployment_id) AND dt.region_id = sqlc.arg(region_id);
