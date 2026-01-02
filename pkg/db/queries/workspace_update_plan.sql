-- name: UpdateWorkspacePlan :exec
UPDATE `workspaces`
SET plan = sqlc.arg(plan)
WHERE id = sqlc.arg(id)
;
