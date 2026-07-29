-- name: ListEnvVarsForRepoConnections :many
SELECT aev.app_id, aev.`key`, aev.value
FROM app_environment_variables aev
INNER JOIN apps a ON aev.app_id = a.id
INNER JOIN environments e ON a.id = e.app_id AND e.id = aev.environment_id
INNER JOIN github_repo_connections gc ON gc.app_id = a.id
WHERE gc.installation_id = sqlc.arg(installation_id)
  AND gc.repository_id = sqlc.arg(repository_id)
  AND e.slug = CASE
    WHEN CAST(sqlc.arg(is_fork_pr) AS SIGNED) = 1 THEN 'preview'
    WHEN sqlc.arg(branch) = CASE
      WHEN gc.default_branch IS NOT NULL AND gc.default_branch <> '' THEN gc.default_branch
      WHEN a.default_branch <> '' THEN a.default_branch
      ELSE 'main'
    END
    THEN 'production'
    ELSE 'preview'
  END;
