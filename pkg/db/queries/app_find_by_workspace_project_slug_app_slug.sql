-- name: FindAppByWorkspaceAndSlugs :one
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT sqlc.embed(p), sqlc.embed(a)
FROM apps a
INNER JOIN projects p ON (a.project_id = p.id COLLATE utf8mb4_0900_ai_ci AND a.project_id = p.id COLLATE utf8mb4_0900_as_cs)
WHERE p.workspace_id = sqlc.arg(workspace_id)
  AND p.slug = sqlc.arg(project_slug)
  AND a.slug = sqlc.arg(app_slug);
