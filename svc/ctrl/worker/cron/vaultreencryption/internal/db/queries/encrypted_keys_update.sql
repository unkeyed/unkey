-- name: UpdateEncryptedKey :execrows
-- UpdateEncryptedKey replaces only the exact ciphertext that was read.
-- Binary comparison preserves concurrent changes on case-insensitive collations.
UPDATE encrypted_keys
SET encrypted = sqlc.arg(ciphertext),
    encryption_key_id = sqlc.arg(key_id)
WHERE pk = sqlc.arg(pk)
  AND workspace_id = sqlc.arg(keyring)
  AND BINARY encrypted = BINARY sqlc.arg(ciphertext_old);
