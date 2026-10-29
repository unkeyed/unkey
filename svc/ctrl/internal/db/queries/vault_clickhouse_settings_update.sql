-- name: UpdateVaultClickhouseSetting :execrows
-- UpdateVaultClickhouseSetting preserves concurrent changes by matching the
-- keyring and old ciphertext using column collations, without changing query limits.
UPDATE clickhouse_workspace_settings
SET password_encrypted = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND workspace_id = sqlc.arg(keyring)
  AND password_encrypted = sqlc.arg(ciphertext_old);
