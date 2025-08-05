-- name: FindGatewayByHostname :one
SELECT hostname, config
FROM gateways
WHERE hostname = ?;
