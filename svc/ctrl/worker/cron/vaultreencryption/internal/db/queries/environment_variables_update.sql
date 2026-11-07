-- name: UpdateEnvironmentVariable :execrows
-- UpdateEnvironmentVariable preserves concurrent changes by matching the
-- keyring and exact old ciphertext, without changing business metadata.
UPDATE app_environment_variables
SET value = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND environment_id = sqlc.arg(keyring)
  AND BINARY value = BINARY sqlc.arg(ciphertext_old);
