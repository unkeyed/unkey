-- name: FindAppAnyById :one
-- Returns an app row by id without filtering on delete_permanently_at.
-- Used by SoftDelete/Restore VOs to load the row regardless of grace
-- state; FindAppById filters scheduled-for-deletion rows.
SELECT *
FROM apps
WHERE id = sqlc.arg(id);
