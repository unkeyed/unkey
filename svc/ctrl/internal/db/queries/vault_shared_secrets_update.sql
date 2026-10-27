-- name: UpdateVaultSharedSecret :execrows
-- UpdateVaultSharedSecret atomically replaces ciphertext and encryption-key
-- metadata only when the keyring and old ciphertext still match byte-for-byte.
UPDATE shared_secrets
SET encrypted = sqlc.arg(ciphertext), encryption_key_id = sqlc.arg(key_id)
WHERE pk = sqlc.arg(pk)
  AND BINARY workspace_id = BINARY sqlc.arg(keyring)
  AND BINARY encrypted = BINARY sqlc.arg(ciphertext_old);
