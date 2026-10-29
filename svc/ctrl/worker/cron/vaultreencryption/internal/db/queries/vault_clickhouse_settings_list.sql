-- name: ListVaultClickhouseSettings :many
-- Scan passwords without parent joins. The primary-key cursor advances
-- past unchanged rows without depending on encryption metadata.
SELECT pk, workspace_id AS keyring, password_encrypted AS ciphertext
FROM clickhouse_workspace_settings
WHERE pk > sqlc.arg(after_pk)
ORDER BY pk
LIMIT ?;
