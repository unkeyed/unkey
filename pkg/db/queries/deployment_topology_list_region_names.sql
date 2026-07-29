-- name: ListDeploymentRegions :many
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT DISTINCT r.name AS region
FROM deployment_topology dt
JOIN regions r ON (dt.region_id COLLATE utf8mb4_0900_ai_ci = r.id AND dt.region_id COLLATE utf8mb4_0900_as_cs = r.id)
WHERE dt.workspace_id = sqlc.arg(workspace_id)
  AND dt.deployment_id = sqlc.arg(deployment_id)
ORDER BY r.name;

-- name: ListDeploymentRegionsByIds :many
SELECT DISTINCT dt.deployment_id AS deployment_id, r.name AS region
FROM deployment_topology dt
JOIN regions r ON (dt.region_id COLLATE utf8mb4_0900_ai_ci = r.id AND dt.region_id COLLATE utf8mb4_0900_as_cs = r.id)
WHERE dt.workspace_id = sqlc.arg(workspace_id)
  AND dt.deployment_id IN (sqlc.slice('deployment_ids'))
ORDER BY dt.deployment_id, r.name;
