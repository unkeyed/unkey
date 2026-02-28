-- name: FindAppWithSettings :one
SELECT
    sqlc.embed(a),
    sqlc.embed(abs),
    sqlc.embed(ars)
FROM apps a
INNER JOIN app_build_settings abs ON abs.app_id = a.id AND abs.environment_id = sqlc.arg(environment_id)
INNER JOIN app_runtime_settings ars ON ars.app_id = a.id AND ars.environment_id = sqlc.arg(environment_id)
WHERE a.project_id = sqlc.arg(project_id)
  AND a.slug = sqlc.arg(slug);
