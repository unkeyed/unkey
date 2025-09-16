-- name: DeleteGatewayByHostname :exec
DELETE FROM gateways WHERE hostname = ?;
