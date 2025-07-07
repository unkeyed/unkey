-- name: UpdateApiDeleteProtection :exec
UPDATE apis
SET delete_protection = sqlc.arg(delete_protection)
WHERE id = sqlc.arg(api_id);
