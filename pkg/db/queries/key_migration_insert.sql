-- name: InsertKeyMigration :exec
INSERT INTO key_migrations (
    id,
    workspace_id,
    algorithm
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(algorithm)
);
