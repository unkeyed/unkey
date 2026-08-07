-- name: ListEnvVarsForRepoConnections :many
SELECT aev.app_id, aev.`key`, aev.value
FROM app_environment_variables aev
INNER JOIN apps a ON aev.app_id = a.id
INNER JOIN environments e ON a.id = e.app_id AND e.id = aev.environment_id
INNER JOIN github_repo_connections gc ON gc.app_id = a.id
WHERE gc.installation_id = sqlc.arg(installation_id)
  AND gc.repository_id = sqlc.arg(repository_id)
  AND CASE
    WHEN CAST(sqlc.arg(is_fork_pr) AS SIGNED) = 1 THEN e.kind = 'preview'
    WHEN sqlc.arg(branch) = COALESCE(NULLIF(gc.default_branch, ''), 'main')
    THEN e.kind = 'production'
    ELSE e.kind = 'preview'
  END;
