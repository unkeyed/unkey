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
INNER JOIN apps a ON (a.id = gc.app_id COLLATE utf8mb4_0900_ai_ci AND a.id = gc.app_id COLLATE utf8mb4_0900_as_cs)
INNER JOIN projects p ON (p.id = gc.project_id COLLATE utf8mb4_0900_ai_ci AND p.id = gc.project_id COLLATE utf8mb4_0900_as_cs)
INNER JOIN environments e ON (e.app_id = a.id COLLATE utf8mb4_0900_ai_ci AND e.app_id = a.id COLLATE utf8mb4_0900_as_cs)
  AND e.slug = CASE
    WHEN CAST(sqlc.arg(is_fork_pr) AS SIGNED) = 1 THEN 'preview'
    WHEN sqlc.arg(branch) = COALESCE(NULLIF(a.default_branch, ''), 'main')
    THEN 'production'
    ELSE 'preview'
  END
INNER JOIN app_build_settings abs ON (abs.app_id = a.id COLLATE utf8mb4_0900_ai_ci AND abs.app_id = a.id COLLATE utf8mb4_0900_as_cs) AND (abs.environment_id = e.id COLLATE utf8mb4_0900_ai_ci AND abs.environment_id = e.id COLLATE utf8mb4_0900_as_cs)
INNER JOIN app_runtime_settings ars ON (ars.app_id = a.id COLLATE utf8mb4_0900_ai_ci AND ars.app_id = a.id COLLATE utf8mb4_0900_as_cs) AND (ars.environment_id = e.id COLLATE utf8mb4_0900_ai_ci AND ars.environment_id = e.id COLLATE utf8mb4_0900_as_cs)
WHERE gc.installation_id = sqlc.arg(installation_id)
  AND gc.repository_id = sqlc.arg(repository_id);
