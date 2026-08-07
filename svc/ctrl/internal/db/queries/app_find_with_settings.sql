-- name: FindAppWithRuntimeSettings :one
SELECT
    sqlc.embed(a),
    sqlc.embed(ars)
FROM apps a
INNER JOIN app_runtime_settings ars ON ars.app_id = a.id AND ars.environment_id = sqlc.arg(environment_id)
WHERE a.id = sqlc.arg(id);
