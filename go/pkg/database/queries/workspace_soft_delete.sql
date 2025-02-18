-- name: SoftDeleteWorkspace :execresult
UPDATE `workspaces`
SET deleted_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
AND delete_protection = false;
