-- name: FindEnvironmentWithSettingsByProjectIdAndSlug :one
SELECT
    sqlc.embed(e),
    bs.dockerfile,
    bs.docker_context,
    rs.port,
    rs.cpu_millicores,
    rs.memory_mib,
    rs.command,
    rs.shutdown_signal,
    rs.healthcheck,
    rs.region_config
FROM environments e
INNER JOIN environment_build_settings bs ON bs.environment_id = e.id
INNER JOIN environment_runtime_settings rs ON rs.environment_id = e.id
WHERE e.workspace_id = sqlc.arg(workspace_id)
  AND e.project_id = sqlc.arg(project_id)
  AND e.slug = sqlc.arg(slug);
