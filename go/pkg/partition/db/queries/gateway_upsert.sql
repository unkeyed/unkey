-- name: UpsertGateway :exec
INSERT INTO gateways (hostname, config)
VALUES (?, ?) ON DUPLICATE KEY UPDATE config = VALUES(config);
