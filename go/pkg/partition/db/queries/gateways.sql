-- name: GetGatewayConfig :one
SELECT hostname, gateway_config
FROM gateways
WHERE hostname = ?;

-- name: UpsertGatewayConfig :exec
INSERT INTO gateways (hostname, gateway_config)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE
    gateway_config = VALUES(gateway_config);


-- name: DeleteGatewayConfig :exec
DELETE FROM gateways
WHERE hostname = ?;
