-- name: SoftDeleteWorkspace :execresult
UPDATE `workspaces`
SET deleted_at = NOW()
WHERE id = sqlc.arg(id)
AND delete_protection = false;
