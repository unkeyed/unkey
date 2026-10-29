-- name: UpdateVaultEncryptedKey :execrows
-- API key plaintext is immutable. The cron serializes re-encryption writes.
-- Keys deleted during re-encryption remain deleted.
UPDATE encrypted_keys
SET encrypted = sqlc.arg(ciphertext),
    encryption_key_id = sqlc.arg(key_id)
WHERE pk = sqlc.arg(pk)
  AND workspace_id = sqlc.arg(keyring);
