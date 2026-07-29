-- name: ListRepoConnectionDeployContexts :many
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT
    sqlc.embed(gc),
    sqlc.embed(p),
    sqlc.embed(e),
    sqlc.embed(a),
    sqlc.embed(abs),
    sqlc.embed(ars)
FROM github_repo_connections gc
INNER JOIN apps a ON (gc.app_id COLLATE utf8mb4_0900_ai_ci = a.id AND gc.app_id COLLATE utf8mb4_0900_as_cs = a.id)
INNER JOIN projects p ON (gc.project_id COLLATE utf8mb4_0900_ai_ci = p.id AND gc.project_id COLLATE utf8mb4_0900_as_cs = p.id)
INNER JOIN environments e ON (a.id COLLATE utf8mb4_0900_ai_ci = e.app_id AND a.id COLLATE utf8mb4_0900_as_cs = e.app_id)
  AND e.slug = CASE
    WHEN CAST(sqlc.arg(is_fork_pr) AS SIGNED) = 1 THEN 'preview'
    WHEN sqlc.arg(branch) = COALESCE(NULLIF(a.default_branch, ''), 'main')
    THEN 'production'
    ELSE 'preview'
  END
INNER JOIN app_build_settings abs ON (a.id COLLATE utf8mb4_0900_ai_ci = abs.app_id AND a.id COLLATE utf8mb4_0900_as_cs = abs.app_id) AND (e.id COLLATE utf8mb4_0900_ai_ci = abs.environment_id AND e.id COLLATE utf8mb4_0900_as_cs = abs.environment_id)
INNER JOIN app_runtime_settings ars ON (a.id COLLATE utf8mb4_0900_ai_ci = ars.app_id AND a.id COLLATE utf8mb4_0900_as_cs = ars.app_id) AND (e.id COLLATE utf8mb4_0900_ai_ci = ars.environment_id AND e.id COLLATE utf8mb4_0900_as_cs = ars.environment_id)
WHERE gc.installation_id = sqlc.arg(installation_id)
  AND gc.repository_id = sqlc.arg(repository_id);
