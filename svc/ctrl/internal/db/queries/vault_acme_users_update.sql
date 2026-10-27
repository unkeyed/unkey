-- name: UpdateVaultAcmeUser :execrows
-- UpdateVaultAcmeUser preserves concurrent changes by matching the keyring and
-- old ciphertext byte-for-byte, without changing ACME registration metadata.
UPDATE acme_users
SET encrypted_key = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND BINARY workspace_id = BINARY sqlc.arg(keyring)
  AND BINARY encrypted_key = BINARY sqlc.arg(ciphertext_old);
