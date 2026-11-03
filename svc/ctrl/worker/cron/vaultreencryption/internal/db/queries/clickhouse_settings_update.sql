-- name: UpdateClickhouseSetting :execrows
-- UpdateClickhouseSetting preserves concurrent changes by matching the
-- keyring and exact old ciphertext, without changing query limits.
UPDATE clickhouse_workspace_settings
SET password_encrypted = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND workspace_id = sqlc.arg(keyring)
  AND BINARY password_encrypted = BINARY sqlc.arg(ciphertext_old);
