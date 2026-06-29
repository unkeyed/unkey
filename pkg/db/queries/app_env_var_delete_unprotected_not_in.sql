-- name: DeleteUnprotectedAppEnvVarsNotInKeys :exec
-- Deletes an environment's unprotected variables whose key is not in the
-- provided set, reconciling toward the desired set while leaving
-- delete-protected variables untouched.
DELETE FROM app_environment_variables
WHERE environment_id = sqlc.arg(environment_id)
  AND (delete_protection = false OR delete_protection IS NULL)
  AND `key` NOT IN (sqlc.slice(env_keys));
