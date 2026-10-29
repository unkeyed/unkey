-- name: UpdateVaultAcmeUser :execrows
-- UpdateVaultAcmeUser preserves concurrent changes by matching the keyring and
-- old ciphertext using column collations, without changing ACME registration metadata.
UPDATE acme_users
SET encrypted_key = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND workspace_id = sqlc.arg(keyring)
  AND encrypted_key = sqlc.arg(ciphertext_old);
