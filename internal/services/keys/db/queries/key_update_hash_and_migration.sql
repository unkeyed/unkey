-- name: UpdateKeyHashAndMigration :exec
-- UpdateKeyHashAndMigration re-hashes a key to SHA-256 after a successful
-- on-demand migration and clears the pending migration marker so future
-- lookups use the standard hash path.
UPDATE `keys`
SET
    hash = sqlc.arg(hash),
    pending_migration_id = sqlc.arg(pending_migration_id),
    start = sqlc.arg(start),
    updated_at_m = sqlc.arg(updated_at_m)
WHERE id = sqlc.arg(id);
