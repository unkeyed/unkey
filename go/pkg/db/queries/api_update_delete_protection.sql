-- name: UpdateApiDeleteProtection :exec
UPDATE apis
SET delete_protection = sqlc.Arg(delete_protection)
WHERE id = sqlc.Arg(api_id)
;
