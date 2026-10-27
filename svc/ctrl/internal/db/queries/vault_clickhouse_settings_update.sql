-- name: UpdateVaultClickhouseSetting :execrows
-- UpdateVaultClickhouseSetting preserves concurrent changes by matching the
-- keyring and old ciphertext byte-for-byte, without changing query limits.
UPDATE clickhouse_workspace_settings
SET password_encrypted = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND BINARY workspace_id = BINARY sqlc.arg(keyring)
  AND BINARY password_encrypted = BINARY sqlc.arg(ciphertext_old);
