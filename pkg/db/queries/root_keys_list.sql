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
    created_at,
    COALESCE(
        (SELECT JSON_ARRAYAGG(p.slug)
        FROM unkey_principal_permissions p
        WHERE p.workspace_id = k.workspace_id
            AND p.principal_type = 'root_key'
            AND p.principal_id = k.id),
        JSON_ARRAY()
    ) AS permissions
FROM unkey_root_keys k
WHERE k.workspace_id = sqlc.arg(workspace_id)
    AND k.deleted_at IS NULL
    AND k.id >= sqlc.arg(id_cursor)
ORDER BY id ASC
LIMIT ?;
