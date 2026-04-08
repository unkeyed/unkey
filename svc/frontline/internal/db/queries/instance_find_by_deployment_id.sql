-- name: FindInstancesByDeploymentID :many
-- FindInstancesByDeploymentID returns all instances for a given deployment
-- with region metadata for instance-aware routing decisions.
SELECT
  i.*,
  r.name AS region_name,
  r.platform AS region_platform
FROM instances i
INNER JOIN regions r ON i.region_id = r.id
WHERE i.deployment_id = sqlc.arg(deployment_id);
