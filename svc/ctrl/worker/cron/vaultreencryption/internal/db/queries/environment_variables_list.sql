-- name: ListEnvironmentVariables :many
-- Scan ciphertext without parent joins. The primary-key cursor advances
-- past unchanged rows without depending on encryption metadata.
SELECT pk, environment_id AS keyring, value AS ciphertext
FROM app_environment_variables
WHERE pk > sqlc.arg(after_pk)
ORDER BY pk
LIMIT ?;
