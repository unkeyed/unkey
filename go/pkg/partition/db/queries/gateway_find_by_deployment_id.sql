-- name: FindGatewayByDeploymentId :one
SELECT hostname, config
FROM gateways
WHERE deployment_id = ?
ORDER BY id DESC
LIMIT 1;
