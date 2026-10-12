-- name: UpdatePermission :exec
UPDATE permissions
SET
    name = sqlc.arg('name'),
    slug = sqlc.arg('slug'),
    description = sqlc.narg('description'),
    updated_at_m = sqlc.arg('updated_at_m')
WHERE workspace_id = sqlc.arg('workspace_id')
  AND id = sqlc.arg('id');
