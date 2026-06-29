-- name: InsertAppEnvironmentVariable :exec
INSERT INTO app_environment_variables (id, workspace_id, app_id, environment_id, `key`, value, `type`, description, delete_protection, created_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(app_id), sqlc.arg(environment_id), sqlc.arg(env_key), sqlc.arg(value), sqlc.arg(type), sqlc.arg(description), sqlc.arg(delete_protection), sqlc.arg(created_at))
ON DUPLICATE KEY UPDATE
  value = VALUES(value),
  `type` = VALUES(`type`),
  description = VALUES(description),
  delete_protection = VALUES(delete_protection),
  updated_at = VALUES(created_at);
