-- name: UpdateVaultDeployment :execrows
-- UpdateVaultDeployment preserves concurrent changes by matching the keyring
-- and entire old blob byte-for-byte, without changing business metadata.
UPDATE deployments
SET encrypted_environment_variables = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND BINARY environment_id = BINARY sqlc.arg(keyring)
  AND BINARY encrypted_environment_variables = BINARY sqlc.arg(ciphertext_old);
