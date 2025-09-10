-- name: UpsertGateway :exec
INSERT INTO gateways (workspace_id, hostname, config)
VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE config = VALUES(config);
