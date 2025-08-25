-- name: FindKeyMigrationByID :one
SELECT
    id,
    workspace_id,
    algorithm
FROM key_migrations
WHERE id = sqlc.arg(id) and workspace_id = sqlc.arg(workspace_id);
