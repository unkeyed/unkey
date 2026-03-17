-- name: InsertAppEnvironmentVariable :exec
INSERT INTO app_environment_variables (id, workspace_id, app_id, environment_id, `key`, value, created_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(app_id), sqlc.arg(environment_id), sqlc.arg(env_key), sqlc.arg(value), sqlc.arg(created_at));
