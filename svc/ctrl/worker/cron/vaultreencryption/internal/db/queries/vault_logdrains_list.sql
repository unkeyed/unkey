-- name: ListVaultLogdrains :many
-- Scan credentials without parent or status filters. The primary-key cursor
-- advances past unchanged rows without depending on encryption metadata.
SELECT pk, workspace_id AS keyring, config AS ciphertext
FROM logdrains
WHERE pk > sqlc.arg(after_pk)
ORDER BY pk
LIMIT ?;
