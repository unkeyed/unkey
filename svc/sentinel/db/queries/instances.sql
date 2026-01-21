-- name: FindInstancesByDeploymentIdAndRegion :many
SELECT
 *
FROM instances
WHERE deployment_id = sqlc.arg(deploymentId) AND region = sqlc.arg(region);
