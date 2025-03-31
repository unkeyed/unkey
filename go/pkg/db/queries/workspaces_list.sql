-- name: ListWorkspaces :many
SELECT
   sqlc.embed(w),
   sqlc.embed(q)
FROM `workspaces` w
LEFT JOIN quota q ON w.id = q.workspace_id
WHERE w.id > sqlc.arg('cursor')
ORDER BY w.id ASC
LIMIT 100;
