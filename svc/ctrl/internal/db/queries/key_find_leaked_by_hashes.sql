-- name: FindLeakedKeysByHashes :many
SELECT k.id, k.hash, w.id AS workspace_id, w.org_id, w.name AS workspace_name, 'rootKey' AS key_type
FROM unkey_root_keys k
JOIN workspaces w ON w.id = k.workspace_id
WHERE k.hash IN (sqlc.slice('hashes'))
  AND k.deleted_at IS NULL
  AND w.deleted_at_m IS NULL
UNION ALL
SELECT k.id, k.hash, w.id AS workspace_id, w.org_id, w.name AS workspace_name,
       IF(k.for_workspace_id IS NULL, 'key', 'rootKey') AS key_type
FROM `keys` k
JOIN workspaces w ON w.id = COALESCE(k.for_workspace_id, k.workspace_id)
WHERE k.hash IN (sqlc.slice('hashes'))
  AND k.deleted_at_m IS NULL
  AND w.deleted_at_m IS NULL
  AND NOT EXISTS (SELECT 1 FROM unkey_root_keys r WHERE r.hash = k.hash);
