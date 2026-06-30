-- name: ListAppEnvVarKeys :many
-- Returns the existing variable keys for an environment. Used to classify a
-- set into created/updated/removed for audit logging.
SELECT `key`
FROM app_environment_variables
WHERE environment_id = sqlc.arg(environment_id);
