-- name: SoftDeleteWorkspace :execresult
UPDATE `workspaces`
SET deleted_at_m = sqlc.arg(now)
WHERE id = sqlc.arg(id)
AND delete_protection = false;
