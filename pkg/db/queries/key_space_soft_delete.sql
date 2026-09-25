-- name: SoftDeleteKeySpace :exec
UPDATE key_auth
SET deleted_at_m = sqlc.arg(now)
WHERE id = sqlc.arg(key_space_id);
