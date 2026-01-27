-- name: ListWorkspacesWithQuotas :many
-- ListWorkspacesWithQuotas returns all enabled workspaces with their quota settings.
-- Used for bulk quota checking operations.
SELECT
   sqlc.embed(w),
   sqlc.embed(q)
FROM `workspaces` w
LEFT JOIN quota q ON w.id = q.workspace_id
WHERE w.enabled = true
ORDER BY w.id ASC;
