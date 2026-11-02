-- name: UpdateEnvironmentVariable :execrows
-- UpdateEnvironmentVariable preserves concurrent changes by matching the
-- keyring and old ciphertext using column collations, without changing business metadata.
UPDATE app_environment_variables
SET value = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND environment_id = sqlc.arg(keyring)
  AND value = sqlc.arg(ciphertext_old);
