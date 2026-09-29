-- name: UpdateUnkeyRootKey :exec
-- UpdateUnkeyRootKey changes mutable fields on a live new-format root key.
UPDATE unkey_root_keys SET
    name = CASE
        WHEN CAST(sqlc.arg(name_specified) AS UNSIGNED) = 1 THEN sqlc.narg(name)
        ELSE name
    END,
    enabled = CASE
        WHEN CAST(sqlc.arg(enabled_specified) AS UNSIGNED) = 1 THEN sqlc.narg(enabled)
        ELSE enabled
    END
WHERE id = sqlc.arg(id)
    AND for_workspace_id = sqlc.arg(for_workspace_id)
    AND deleted_at IS NULL;

-- name: UpdateLegacyRootKey :exec
-- UpdateLegacyRootKey changes mutable fields on a live legacy root key.
UPDATE `keys` SET
    name = CASE
        WHEN CAST(sqlc.arg(name_specified) AS UNSIGNED) = 1 THEN sqlc.narg(name)
        ELSE name
    END,
    enabled = CASE
        WHEN CAST(sqlc.arg(enabled_specified) AS UNSIGNED) = 1 THEN sqlc.narg(enabled)
        ELSE enabled
    END,
    updated_at_m = sqlc.arg(now)
WHERE id = sqlc.arg(id)
    AND for_workspace_id = sqlc.arg(for_workspace_id)
    AND deleted_at_m IS NULL;
