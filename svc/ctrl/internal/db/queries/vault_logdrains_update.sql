-- name: UpdateVaultLogdrain :execrows
-- UpdateVaultLogdrain preserves concurrent changes by matching the keyring and
-- entire old config byte-for-byte, without changing lease or delivery metadata.
UPDATE logdrains
SET config = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND BINARY workspace_id = BINARY sqlc.arg(keyring)
  AND BINARY config = BINARY sqlc.arg(ciphertext_old);
