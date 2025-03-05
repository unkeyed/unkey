-- name: UpdateWorkspaceEnabled :execresult
UPDATE `workspaces`
SET enabled = sqlc.arg(enabled)
WHERE id = sqlc.arg(id)
;
