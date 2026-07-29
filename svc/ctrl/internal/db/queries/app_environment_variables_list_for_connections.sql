-- name: ListEnvVarsForRepoConnections :many
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT aev.app_id, aev.`key`, aev.value
FROM app_environment_variables aev
INNER JOIN apps a ON (aev.app_id = a.id COLLATE utf8mb4_0900_ai_ci AND aev.app_id = a.id COLLATE utf8mb4_0900_as_cs)
INNER JOIN environments e ON (a.id COLLATE utf8mb4_0900_ai_ci = e.app_id AND a.id COLLATE utf8mb4_0900_as_cs = e.app_id) AND (e.id COLLATE utf8mb4_0900_ai_ci = aev.environment_id AND e.id COLLATE utf8mb4_0900_as_cs = aev.environment_id)
INNER JOIN github_repo_connections gc ON (gc.app_id COLLATE utf8mb4_0900_ai_ci = a.id AND gc.app_id COLLATE utf8mb4_0900_as_cs = a.id)
WHERE gc.installation_id = sqlc.arg(installation_id)
  AND gc.repository_id = sqlc.arg(repository_id)
  AND e.slug = CASE
    WHEN CAST(sqlc.arg(is_fork_pr) AS SIGNED) = 1 THEN 'preview'
    WHEN sqlc.arg(branch) = COALESCE(NULLIF(a.default_branch, ''), 'main')
    THEN 'production'
    ELSE 'preview'
  END;
