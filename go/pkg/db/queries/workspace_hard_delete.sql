-- name: HardDeleteWorkspace :execresult
DELETE FROM `workspaces`
WHERE id = sqlc.arg(id)
AND delete_protection = false;
