-- name: UpdateLogdrain :execrows
-- UpdateLogdrain preserves concurrent changes by matching the keyring using
-- its column collation and the entire old config, without changing lease or delivery metadata.
UPDATE logdrains
SET config = sqlc.arg(ciphertext)
WHERE pk = sqlc.arg(pk)
  AND workspace_id = sqlc.arg(keyring)
  AND config = sqlc.arg(ciphertext_old);
