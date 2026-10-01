-- name: MarkAppDeleting :exec
UPDATE apps SET deleting_at = COALESCE(deleting_at, sqlc.arg(deleting_at))
WHERE id = sqlc.arg(id);
