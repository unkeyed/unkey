-- name: ListVaultSharedSecrets :many
-- Include expired secrets and rows without parents. The primary-key cursor
-- advances past unchanged rows without depending on encryption metadata.
SELECT pk, workspace_id AS keyring, encrypted AS ciphertext
FROM shared_secrets
WHERE pk > sqlc.arg(after_pk)
ORDER BY pk
LIMIT ?;
