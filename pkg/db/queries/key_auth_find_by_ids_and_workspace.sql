-- name: FindKeyAuthsByIdsAndWorkspace :many
-- Returns the subset of the given keyspace ids that exist in this workspace
-- and are not soft-deleted.
SELECT id FROM key_auth
WHERE workspace_id = sqlc.arg(workspace_id)
  AND id IN (sqlc.slice(key_auth_ids))
  AND deleted_at_m IS NULL;
