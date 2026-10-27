-- name: ListVaultDeployments :many
-- Scan snapshots without parent or status filters. The primary-key cursor
-- advances past unchanged rows without depending on encryption metadata.
SELECT pk, environment_id AS keyring, encrypted_environment_variables AS ciphertext
FROM deployments
WHERE pk > sqlc.arg(after_pk)
ORDER BY pk
LIMIT ?;
