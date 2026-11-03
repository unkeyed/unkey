-- name: UpdateCertificate :execrows
-- UpdateCertificate preserves concurrent changes by matching the keyring
-- and exact old ciphertext, without changing certificate metadata.
UPDATE certificates
SET encrypted_private_key = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND workspace_id = sqlc.arg(keyring)
  AND BINARY encrypted_private_key = BINARY sqlc.arg(ciphertext_old);
