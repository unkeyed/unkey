-- name: ListDesiredDeploymentTopology :many
-- ListDesiredDeploymentTopology returns all deployment topologies matching the desired state for a region.
-- Used during bootstrap to stream all running deployments to krane.
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
    sqlc.embed(dt),
    sqlc.embed(d),
    w.k8s_namespace
FROM `deployment_topology` dt
INNER JOIN `deployments` d ON (dt.deployment_id COLLATE utf8mb4_0900_ai_ci = d.id AND dt.deployment_id COLLATE utf8mb4_0900_as_cs = d.id)
INNER JOIN `workspaces` w ON (d.workspace_id COLLATE utf8mb4_0900_ai_ci = w.id AND d.workspace_id COLLATE utf8mb4_0900_as_cs = w.id)
INNER JOIN `regions` r ON (dt.region_id COLLATE utf8mb4_0900_ai_ci = r.id AND dt.region_id COLLATE utf8mb4_0900_as_cs = r.id)
WHERE (sqlc.arg(region) = '' OR r.name = sqlc.arg(region))
    AND d.desired_state = sqlc.arg(desired_state)
    AND dt.deployment_id > sqlc.arg(pagination_cursor)
ORDER BY dt.deployment_id ASC
LIMIT ?;
