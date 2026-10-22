-- name: FindLeakedKeysByHashes :many
SELECT k.id, k.hash, w.org_id, w.name AS workspace_name
FROM unkey_root_keys k
JOIN workspaces w ON w.id = k.workspace_id
WHERE k.hash IN (sqlc.slice('hashes'))
  AND k.deleted_at IS NULL
  AND w.deleted_at_m IS NULL
UNION ALL
SELECT k.id, k.hash, w.org_id, w.name AS workspace_name
FROM `keys` k
JOIN workspaces w ON w.id = COALESCE(k.for_workspace_id, k.workspace_id)
WHERE k.hash IN (sqlc.slice('hashes'))
  AND k.deleted_at_m IS NULL
  AND w.deleted_at_m IS NULL
  AND NOT EXISTS (SELECT 1 FROM unkey_root_keys r WHERE r.hash = k.hash);
