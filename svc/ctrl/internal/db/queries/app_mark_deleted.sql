-- name: MarkAppDeleted :exec
UPDATE apps SET deleted_at = COALESCE(deleted_at, sqlc.arg(deleted_at))
WHERE id = sqlc.arg(id);
