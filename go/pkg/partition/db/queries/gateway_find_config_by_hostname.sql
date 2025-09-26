-- name: FindGatewayByHostname :one
SELECT hostname, config, workspace_id
FROM gateways
WHERE hostname = ?;
