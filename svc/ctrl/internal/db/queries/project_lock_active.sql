-- name: LockActiveProject :one
SELECT id FROM projects
WHERE id = sqlc.arg(id) AND deleting_at IS NULL
LOCK IN SHARE MODE;
