-- name: UpdateVaultCertificate :execrows
-- UpdateVaultCertificate preserves concurrent changes by matching the keyring
-- and old ciphertext using column collations, without changing certificate metadata.
UPDATE certificates
SET encrypted_private_key = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND workspace_id = sqlc.arg(keyring)
  AND encrypted_private_key = sqlc.arg(ciphertext_old);
