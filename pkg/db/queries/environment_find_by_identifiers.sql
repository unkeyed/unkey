-- name: FindEnvironmentByIdentifiers :one
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT e.*
FROM environments e
JOIN apps a ON (e.app_id COLLATE utf8mb4_0900_ai_ci = a.id AND e.app_id COLLATE utf8mb4_0900_as_cs = a.id) AND (e.workspace_id COLLATE utf8mb4_0900_ai_ci = a.workspace_id AND e.workspace_id COLLATE utf8mb4_0900_as_cs = a.workspace_id)
JOIN projects p ON (a.project_id COLLATE utf8mb4_0900_ai_ci = p.id AND a.project_id COLLATE utf8mb4_0900_as_cs = p.id) AND (a.workspace_id COLLATE utf8mb4_0900_ai_ci = p.workspace_id AND a.workspace_id COLLATE utf8mb4_0900_as_cs = p.workspace_id)
WHERE e.workspace_id = sqlc.arg(workspace_id)
  AND (p.id = sqlc.arg(project) OR p.slug = sqlc.arg(project))
  AND (a.id = sqlc.arg(app) OR a.slug = sqlc.arg(app))
  AND (e.id = sqlc.arg(environment) OR e.slug = sqlc.arg(environment))
LIMIT 1;
