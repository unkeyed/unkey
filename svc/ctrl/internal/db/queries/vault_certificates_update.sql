-- name: UpdateVaultCertificate :execrows
-- UpdateVaultCertificate preserves concurrent changes by matching the keyring
-- and old ciphertext byte-for-byte, without changing certificate metadata.
UPDATE certificates
SET encrypted_private_key = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND BINARY workspace_id = BINARY sqlc.arg(keyring)
  AND BINARY encrypted_private_key = BINARY sqlc.arg(ciphertext_old);
