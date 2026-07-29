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
INNER JOIN `deployments` d ON (d.id = dt.deployment_id COLLATE utf8mb4_0900_ai_ci AND d.id = dt.deployment_id COLLATE utf8mb4_0900_as_cs)
INNER JOIN `workspaces` w ON (w.id = d.workspace_id COLLATE utf8mb4_0900_ai_ci AND w.id = d.workspace_id COLLATE utf8mb4_0900_as_cs)
INNER JOIN `regions` r ON (r.id = dt.region_id COLLATE utf8mb4_0900_ai_ci AND r.id = dt.region_id COLLATE utf8mb4_0900_as_cs)
INNER JOIN `environments` e ON (e.id = d.environment_id COLLATE utf8mb4_0900_ai_ci AND e.id = d.environment_id COLLATE utf8mb4_0900_as_cs)
LEFT JOIN `github_repo_connections` grc ON (grc.app_id = d.app_id COLLATE utf8mb4_0900_ai_ci AND grc.app_id = d.app_id COLLATE utf8mb4_0900_as_cs)
WHERE dt.deployment_id = sqlc.arg(deployment_id) AND dt.region_id = sqlc.arg(region_id)
LIMIT 1;
