-- name: UpdateVaultEnvironmentVariable :execrows
-- UpdateVaultEnvironmentVariable preserves concurrent changes by matching the
-- keyring and old ciphertext byte-for-byte, without changing business metadata.
UPDATE app_environment_variables
SET value = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND BINARY environment_id = BINARY sqlc.arg(keyring)
  AND BINARY value = BINARY sqlc.arg(ciphertext_old);
