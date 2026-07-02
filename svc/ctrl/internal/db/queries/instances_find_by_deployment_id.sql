-- name: FindInstancesByDeploymentId :many
SELECT
 *
FROM instances
WHERE deployment_id = sqlc.arg(deploymentId);
