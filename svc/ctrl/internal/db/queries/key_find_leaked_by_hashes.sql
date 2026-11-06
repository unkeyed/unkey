-- name: FindLeakedKeysByHashes :many
SELECT k.id, k.hash, w.id AS workspace_id, w.org_id, w.name AS workspace_name, 'unkey_root_keys' AS store, 'root' AS kind
FROM unkey_root_keys k
JOIN workspaces w ON w.id = k.workspace_id
WHERE k.hash IN (sqlc.slice('hashes'))
  AND k.deleted_at IS NULL
  AND w.deleted_at_m IS NULL
UNION ALL
SELECT k.id, k.hash, w.id AS workspace_id, w.org_id, w.name AS workspace_name, 'keys' AS store,
  CASE WHEN k.for_workspace_id IS NOT NULL THEN 'root' ELSE 'api' END AS kind
FROM `keys` k
JOIN workspaces w ON w.id = COALESCE(k.for_workspace_id, k.workspace_id)
WHERE k.hash IN (sqlc.slice('hashes'))
  AND k.deleted_at_m IS NULL
  AND w.deleted_at_m IS NULL
  AND NOT EXISTS (SELECT 1 FROM unkey_root_keys r WHERE r.hash = k.hash);
