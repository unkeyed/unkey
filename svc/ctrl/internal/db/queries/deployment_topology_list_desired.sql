-- name: ListDesiredDeploymentTopology :many
-- ListDesiredDeploymentTopology returns all deployment topologies matching the desired state for a region.
-- Used during bootstrap to stream all running deployments to krane.
SELECT
    sqlc.embed(dt),
    sqlc.embed(d),
    w.k8s_namespace
FROM `deployment_topology` dt
INNER JOIN `deployments` d ON dt.deployment_id = d.id
INNER JOIN `workspaces` w ON d.workspace_id = w.id
INNER JOIN `regions` r ON dt.region_id = r.id
WHERE (sqlc.arg(region) = '' OR r.name = sqlc.arg(region))
    AND d.desired_state = sqlc.arg(desired_state)
    AND dt.deployment_id > sqlc.arg(pagination_cursor)
ORDER BY dt.deployment_id ASC
LIMIT ?;
