-- name: ListWorkspaces :many
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
   sqlc.embed(w),
   sqlc.embed(q)
FROM `workspaces` w
LEFT JOIN quota q ON (w.id COLLATE utf8mb4_0900_ai_ci = q.workspace_id AND w.id COLLATE utf8mb4_0900_as_cs = q.workspace_id)
WHERE w.id > sqlc.arg('cursor')
ORDER BY w.id ASC
LIMIT 100;
