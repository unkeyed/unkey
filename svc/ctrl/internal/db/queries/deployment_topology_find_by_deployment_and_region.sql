-- name: FindDeploymentTopologyByDeploymentAndRegion :one
-- FindDeploymentTopologyByDeploymentAndRegion returns a single deployment topology with all
-- joined data needed for the Watch stream. Used by the unified WatchDeploymentChanges RPC.
SELECT
    sqlc.embed(dt),
    sqlc.embed(d),
    w.k8s_namespace,
    e.slug AS environment_slug,
    r.name AS region_name,
    grc.repository_full_name AS git_repo
FROM `deployment_topology` dt
INNER JOIN `deployments` d ON d.id = dt.deployment_id
INNER JOIN `workspaces` w ON w.id = d.workspace_id
INNER JOIN `regions` r ON r.id = dt.region_id
INNER JOIN `environments` e ON e.id = d.environment_id
LEFT JOIN `github_repo_connections` grc ON grc.app_id = d.app_id
WHERE dt.deployment_id = sqlc.arg(deployment_id) AND dt.region_id = sqlc.arg(region_id)
LIMIT 1;
