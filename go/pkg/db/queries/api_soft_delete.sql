-- name: SoftDeleteApi :exec
UPDATE apis
SET deleted_at_m = sqlc.Arg(now)
WHERE id = sqlc.Arg(api_id);
