-- name: ListDeletionsByWorkspace :many
SELECT *
FROM `deletions`
WHERE workspace_id = sqlc.arg(workspace_id)
ORDER BY delete_permanently_at ASC;
