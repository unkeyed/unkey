-- name: UpdateKeyHashAndMigration :exec
UPDATE `keys`
SET 
    hash = sqlc.arg(hash),
    pending_migration_id = sqlc.arg(pending_migration_id),
    updated_at_m = sqlc.arg(updated_at_m)
WHERE id = sqlc.arg(id);