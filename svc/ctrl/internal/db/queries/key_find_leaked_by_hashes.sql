-- name: FindLeakedKeysByHashes :many
SELECT k.id, k.hash, w.org_id, w.name AS workspace_name
FROM `keys` k
JOIN workspaces w ON w.id = k.for_workspace_id
WHERE k.hash IN (sqlc.slice('hashes'))
  AND k.deleted_at_m IS NULL
  AND w.deleted_at_m IS NULL
  AND w.org_id <> '';
