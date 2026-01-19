-- name: FindDeploymentTopologyByVersions :many
-- FindDeploymentTopologyByVersions returns deployment topologies for specific versions.
-- Used after ListClusterStateVersions to hydrate the full deployment data.
SELECT
    sqlc.embed(dt),
    sqlc.embed(d),
    w.k8s_namespace
FROM `deployment_topology` dt
INNER JOIN `deployments` d ON dt.deployment_id = d.id
INNER JOIN `workspaces` w ON d.workspace_id = w.id
WHERE dt.version IN (sqlc.slice(versions));
