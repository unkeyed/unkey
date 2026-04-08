-- name: FindKeyMigrationByID :one
-- FindKeyMigrationByID returns the migration record for a key, scoped to
-- the given workspace. The algorithm field determines which hashing scheme
-- the caller should use for the initial lookup before migrating to SHA-256.
SELECT
    id,
    workspace_id,
    algorithm
FROM key_migrations
WHERE id = sqlc.arg(id)
and workspace_id = sqlc.arg(workspace_id);
