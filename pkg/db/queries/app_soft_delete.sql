-- name: SoftDeleteApp :exec
UPDATE apps
SET deleted_at_m = sqlc.Arg(now)
WHERE id = sqlc.Arg(app_id);
