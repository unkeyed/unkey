-- name: FindInstancesByDeploymentID :many
-- FindInstancesByDeploymentID returns all instances for a given deployment
-- with region metadata for instance-aware routing decisions.
SELECT
  i.id,
  i.workspace_id,
  i.project_id,
  i.app_id,
  i.address,
  i.status,
  r.name AS region_name,
  r.platform AS region_platform
FROM instances i
INNER JOIN regions r ON r.id = i.region_id
WHERE i.deployment_id = sqlc.arg(deployment_id);
