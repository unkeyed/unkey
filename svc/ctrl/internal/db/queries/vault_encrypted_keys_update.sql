-- name: UpdateVaultEncryptedKey :execrows
-- Replace only the version that was read. Binary comparison protects
-- concurrent writes even on case-insensitive database collations.
UPDATE encrypted_keys
SET encrypted = sqlc.arg(ciphertext), encryption_key_id = sqlc.arg(key_id)
WHERE pk = sqlc.arg(pk)
  AND BINARY workspace_id = BINARY sqlc.arg(keyring)
  AND BINARY encrypted = BINARY sqlc.arg(ciphertext_old);
