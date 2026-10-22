-- name: UpdateAppConnectionName :exec
UPDATE app_connections
SET name = sqlc.arg(name), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
