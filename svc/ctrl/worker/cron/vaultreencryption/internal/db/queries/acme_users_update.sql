-- name: UpdateAcmeUser :execrows
-- UpdateAcmeUser preserves concurrent changes by matching the keyring and
-- exact old ciphertext, without changing ACME registration metadata.
UPDATE acme_users
SET encrypted_key = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND workspace_id = sqlc.arg(keyring)
  AND BINARY encrypted_key = BINARY sqlc.arg(ciphertext_old);
