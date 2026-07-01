-- name: ListRepoConnectionDeployContexts :many
SELECT
    sqlc.embed(gc),
    sqlc.embed(p),
    sqlc.embed(e),
    sqlc.embed(a),
    sqlc.embed(abs),
    sqlc.embed(ars)
FROM github_repo_connections gc
INNER JOIN apps a ON a.id = gc.app_id
INNER JOIN projects p ON p.id = gc.project_id
INNER JOIN environments e ON e.app_id = a.id
  AND e.slug = CASE
    WHEN CAST(sqlc.arg(is_fork_pr) AS SIGNED) = 1 THEN 'preview'
    WHEN sqlc.arg(branch) = COALESCE(NULLIF(a.default_branch, ''), 'main')
    THEN 'production'
    ELSE 'preview'
  END
INNER JOIN app_build_settings abs ON abs.app_id = a.id AND abs.environment_id = e.id
INNER JOIN app_runtime_settings ars ON ars.app_id = a.id AND ars.environment_id = e.id
WHERE gc.installation_id = sqlc.arg(installation_id)
  AND gc.repository_id = sqlc.arg(repository_id);
