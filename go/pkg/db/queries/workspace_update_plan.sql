-- name: UpdateWorkspacePlan :execresult
UPDATE `workspaces`
SET plan = sqlc.arg(plan)
WHERE id = sqlc.arg(id)
;
