-- name: ListRepoConnectionDeployContexts :many
SELECT
    p.id AS project_id,
    e.id AS environment_id,
    a.id AS app_id,
    abs.auto_deploy AS build_settings_auto_deploy,
    abs.watch_paths AS build_settings_watch_paths
FROM github_repo_connections gc
INNER JOIN apps a ON a.id = gc.app_id
INNER JOIN projects p ON p.id = gc.project_id
INNER JOIN environments e ON e.app_id = a.id
  AND CASE
    WHEN CAST(sqlc.arg(is_fork_pr) AS SIGNED) = 1 THEN e.kind = 'preview'
    WHEN sqlc.arg(branch) = COALESCE(NULLIF(gc.default_branch, ''), 'main')
    THEN e.kind = 'production'
    ELSE e.kind = 'preview'
  END
INNER JOIN app_build_settings abs ON abs.app_id = a.id AND abs.environment_id = e.id
INNER JOIN app_runtime_settings ars ON ars.app_id = a.id AND ars.environment_id = e.id
WHERE gc.installation_id = sqlc.arg(installation_id)
  AND gc.repository_id = sqlc.arg(repository_id);
