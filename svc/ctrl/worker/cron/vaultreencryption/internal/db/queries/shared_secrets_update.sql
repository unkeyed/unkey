-- name: UpdateSharedSecret :execrows
-- UpdateSharedSecret atomically replaces ciphertext and encryption-key
-- metadata only when the keyring and exact old ciphertext still match.
UPDATE shared_secrets
SET encrypted = sqlc.arg(ciphertext), encryption_key_id = sqlc.arg(key_id)
WHERE pk = sqlc.arg(pk)
  AND workspace_id = sqlc.arg(keyring)
  AND BINARY encrypted = BINARY sqlc.arg(ciphertext_old);
