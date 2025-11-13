-- name: UpsertInstance :exec
INSERT INTO instance (id, deployment_id, status, config)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  deployment_id = VALUES(deployment_id),
  status = VALUES(status),
  config = VALUES(config);
