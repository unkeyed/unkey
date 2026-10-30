-- name: ListVaultAcmeUsers :many
-- Scan account keys without parent joins. The primary-key cursor advances
-- past unchanged rows without depending on encryption metadata.
SELECT pk, workspace_id AS keyring, encrypted_key AS ciphertext
FROM acme_users
WHERE pk > sqlc.arg(after_pk)
ORDER BY pk
LIMIT ?;
