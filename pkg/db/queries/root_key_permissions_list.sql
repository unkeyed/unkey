-- name: ListRootKeyPermissions :many
-- ListRootKeyPermissions loads principal permissions for a page of new root keys.
-- Callers deduplicate exact strings, not collation-equivalent strings.
SELECT
    principal_id AS key_id,
    slug
FROM unkey_principal_permissions
WHERE workspace_id = sqlc.arg(workspace_id)
    AND principal_type = 'root_key'
    AND principal_id IN (sqlc.slice(key_ids));
