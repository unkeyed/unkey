-- name: DeleteUnprotectedAppEnvVarsByEnvironmentId :exec
-- Deletes all unprotected variables for an environment, leaving
-- delete-protected variables untouched. Used when the desired set is empty.
DELETE FROM app_environment_variables
WHERE environment_id = sqlc.arg(environment_id)
  AND (delete_protection = false OR delete_protection IS NULL);
