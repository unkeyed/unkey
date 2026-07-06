-- name: UpdateApp :exec
UPDATE apps a
SET
    name = CASE
        WHEN CAST(sqlc.arg('name_specified') AS UNSIGNED) = 1 THEN sqlc.arg('name')
        ELSE a.name
    END,
    slug = CASE
        WHEN CAST(sqlc.arg('slug_specified') AS UNSIGNED) = 1 THEN sqlc.arg('slug')
        ELSE a.slug
    END,
    default_branch = CASE
        WHEN CAST(sqlc.arg('default_branch_specified') AS UNSIGNED) = 1 THEN sqlc.arg('default_branch')
        ELSE a.default_branch
    END,
    delete_protection = CASE
        WHEN CAST(sqlc.arg('delete_protection_specified') AS UNSIGNED) = 1 THEN sqlc.narg('delete_protection')
        ELSE a.delete_protection
    END,
    updated_at = sqlc.arg('updated_at')
WHERE workspace_id = sqlc.arg('workspace_id')
  AND id = sqlc.arg('id');
