-- name: ListDesiredDeploymentTopology :many
-- ListDesiredDeploymentTopology returns all deployment topologies matching the desired state for a region.
-- Used during bootstrap to stream all running deployments to krane.
-- The version parameter is deprecated and ignored (kept for backwards compatibility).
SELECT
    sqlc.embed(dt),
    sqlc.embed(d),
    w.k8s_namespace
FROM `deployment_topology` dt
INNER JOIN `deployments` d ON dt.deployment_id = d.id
INNER JOIN `workspaces` w ON d.workspace_id = w.id
WHERE (sqlc.arg(region) = '' OR dt.region = sqlc.arg(region))
    AND d.desired_state = sqlc.arg(desired_state)
    AND dt.deployment_id > sqlc.arg(pagination_cursor)
ORDER BY dt.deployment_id ASC
LIMIT ?;
