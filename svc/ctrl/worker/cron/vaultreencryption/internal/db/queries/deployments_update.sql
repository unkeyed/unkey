-- name: UpdateDeployment :execrows
-- UpdateDeployment preserves concurrent changes by matching the keyring
-- using its column collation and the entire old blob, without changing business metadata.
UPDATE deployments
SET encrypted_environment_variables = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND environment_id = sqlc.arg(keyring)
  AND encrypted_environment_variables = sqlc.arg(ciphertext_old);
