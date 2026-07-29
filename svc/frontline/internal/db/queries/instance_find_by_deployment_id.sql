-- name: FindInstancesByDeploymentID :many
-- FindInstancesByDeploymentID returns all instances for a given deployment
-- with region metadata for instance-aware routing decisions.
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
  i.*,
  r.name AS region_name,
  r.platform AS region_platform
FROM instances i
INNER JOIN regions r ON (r.id = i.region_id COLLATE utf8mb4_0900_ai_ci AND r.id = i.region_id COLLATE utf8mb4_0900_as_cs)
WHERE i.deployment_id = sqlc.arg(deployment_id);
