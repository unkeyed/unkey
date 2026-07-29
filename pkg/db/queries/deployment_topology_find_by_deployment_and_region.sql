-- name: FindDeploymentTopologyByDeploymentAndRegion :one
-- FindDeploymentTopologyByDeploymentAndRegion returns a single deployment topology with all
-- joined data needed for the Watch stream. Used by the unified WatchDeploymentChanges RPC.
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
    sqlc.embed(dt),
    sqlc.embed(d),
    w.k8s_namespace,
    e.slug AS environment_slug,
    r.name AS region_name,
    grc.repository_full_name AS git_repo
FROM `deployment_topology` dt
INNER JOIN `deployments` d ON (dt.deployment_id COLLATE utf8mb4_0900_ai_ci = d.id AND dt.deployment_id COLLATE utf8mb4_0900_as_cs = d.id)
INNER JOIN `workspaces` w ON (d.workspace_id COLLATE utf8mb4_0900_ai_ci = w.id AND d.workspace_id COLLATE utf8mb4_0900_as_cs = w.id)
INNER JOIN `regions` r ON (dt.region_id COLLATE utf8mb4_0900_ai_ci = r.id AND dt.region_id COLLATE utf8mb4_0900_as_cs = r.id)
INNER JOIN `environments` e ON (d.environment_id COLLATE utf8mb4_0900_ai_ci = e.id AND d.environment_id COLLATE utf8mb4_0900_as_cs = e.id)
LEFT JOIN `github_repo_connections` grc ON (d.app_id COLLATE utf8mb4_0900_ai_ci = grc.app_id AND d.app_id COLLATE utf8mb4_0900_as_cs = grc.app_id)
WHERE dt.deployment_id = sqlc.arg(deployment_id) AND dt.region_id = sqlc.arg(region_id)
LIMIT 1;
