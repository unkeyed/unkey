-- name: ListDeploymentTopologyByRegion :many
-- ListDeploymentTopologyByRegion returns deployment topologies for a region with version > after_version.
-- Used by WatchDeployments to stream deployment state changes to krane agents.
SELECT
    sqlc.embed(dt),
    sqlc.embed(d),
    w.k8s_namespace
FROM `deployment_topology` dt
INNER JOIN `deployments` d ON dt.deployment_id = d.id
INNER JOIN `workspaces` w ON d.workspace_id = w.id
WHERE dt.region = sqlc.arg(region) AND dt.version > sqlc.arg(afterVersion)
ORDER BY dt.version ASC
LIMIT ?;
