-- name: FindAppByProjectAndIdOrSlug :one
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT a.*
FROM apps a
JOIN projects p ON (a.project_id COLLATE utf8mb4_0900_ai_ci = p.id AND a.project_id COLLATE utf8mb4_0900_as_cs = p.id) AND (a.workspace_id COLLATE utf8mb4_0900_ai_ci = p.workspace_id AND a.workspace_id COLLATE utf8mb4_0900_as_cs = p.workspace_id)
WHERE a.workspace_id = sqlc.arg(workspace_id)
  AND (p.id = sqlc.arg(project) OR p.slug = sqlc.arg(project))
  AND (a.id = sqlc.arg(app) OR a.slug = sqlc.arg(app))
LIMIT 1;
