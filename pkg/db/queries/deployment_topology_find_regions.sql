-- name: FindDeploymentRegions :many
-- Returns all regions where a deployment is configured.
-- Used for fan-out: when a deployment changes, emit state_change to each region.
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT r.*
FROM `deployment_topology` dt
INNER JOIN `regions` r ON (dt.region_id COLLATE utf8mb4_0900_ai_ci = r.id AND dt.region_id COLLATE utf8mb4_0900_as_cs = r.id)
WHERE dt.deployment_id = sqlc.arg(deployment_id);
