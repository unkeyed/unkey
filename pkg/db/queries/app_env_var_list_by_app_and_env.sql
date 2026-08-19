-- name: ListAppEnvVarsByAppAndEnv :many
SELECT id, `key`, value, `type`, description, created_at
FROM app_environment_variables
WHERE app_id = sqlc.arg(app_id)
  AND environment_id = sqlc.arg(environment_id)
  AND id >= sqlc.arg(id_cursor)
ORDER BY id ASC
LIMIT ?;
