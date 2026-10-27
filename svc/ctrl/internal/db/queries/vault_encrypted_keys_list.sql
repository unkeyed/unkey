-- name: ListVaultEncryptedKeys :many
-- Scan ciphertext without parent joins. The primary-key cursor advances
-- past unchanged rows without depending on encryption metadata.
SELECT pk, workspace_id AS keyring, encrypted AS ciphertext
FROM encrypted_keys
WHERE pk > sqlc.arg(after_pk)
ORDER BY pk
LIMIT ?;
