-- name: ListRootKeys :many
-- ListRootKeys returns live root keys from the new store for one customer workspace.
-- The cursor is inclusive: a cursor of key_b returns key_b before key_c.
SELECT
    id,
    name,
    prefix,
    start,
    end,
    enabled,
    expires,
    created_at
FROM unkey_root_keys
WHERE workspace_id = sqlc.arg(workspace_id)
    AND deleted_at IS NULL
    AND id >= sqlc.arg(id_cursor)
ORDER BY id ASC
LIMIT ?;
