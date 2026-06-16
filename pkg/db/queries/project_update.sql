-- name: UpdateProject :exec
UPDATE projects p
SET
    name = CASE
        WHEN CAST(sqlc.arg('name_specified') AS UNSIGNED) = 1 THEN sqlc.arg('name')
        ELSE p.name
    END,
    delete_protection = CASE
        WHEN CAST(sqlc.arg('delete_protection_specified') AS UNSIGNED) = 1 THEN sqlc.narg('delete_protection')
        ELSE p.delete_protection
    END,
    updated_at = sqlc.arg('updated_at')
WHERE workspace_id = sqlc.arg('workspace_id')
  AND slug = sqlc.arg('slug');
