-- name: FindEnabledKeysByHashesForUpdate :many
SELECT id, hash, workspace_id, for_workspace_id
FROM `keys`
WHERE hash IN (sqlc.slice('hashes'))
  AND enabled = TRUE
  AND deleted_at_m IS NULL
ORDER BY hash
FOR UPDATE;
