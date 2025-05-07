-- name: UpdateKeySpaceSize :exec
UPDATE key_auth
SET
  size_approx = sqlc.arg(size_approx),
  size_last_updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(key_auth_id);
