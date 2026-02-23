-- name: FindEnvironmentWithSettingsByProjectIdAndSlug :one
SELECT
    sqlc.embed(e),
    sqlc.embed(ebs),
    sqlc.embed(ers)
FROM environments e
INNER JOIN environment_build_settings ebs ON ebs.environment_id = e.id
INNER JOIN environment_runtime_settings ers ON ers.environment_id = e.id
WHERE e.workspace_id = sqlc.arg(workspace_id)
  AND e.project_id = sqlc.arg(project_id)
  AND e.slug = sqlc.arg(slug);
