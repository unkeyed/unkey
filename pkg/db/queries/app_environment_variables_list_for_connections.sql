-- name: ListEnvVarsForRepoConnections :many
SELECT aev.app_id, aev.`key`, aev.value
FROM app_environment_variables aev
INNER JOIN apps a ON a.id = aev.app_id AND a.environment_id = aev.environment_id
INNER JOIN github_repo_connections gc ON gc.app_id = a.id
INNER JOIN projects p ON p.id = gc.project_id
INNER JOIN environments e ON e.id = a.environment_id
WHERE gc.installation_id = sqlc.arg(installation_id)
  AND gc.repository_id = sqlc.arg(repository_id)
  AND e.slug = CASE
    WHEN sqlc.arg(branch) = COALESCE(NULLIF(p.default_branch, ''), 'main')
    THEN 'production'
    ELSE 'preview'
  END;
