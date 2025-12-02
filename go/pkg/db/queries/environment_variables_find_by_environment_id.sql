-- name: FindEnvironmentVariablesByEnvironmentId :many
SELECT `key`, value
FROM environment_variables
WHERE environment_id = sqlc.arg(environment_id)
  AND deleted_at IS NULL;
